package engine

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/FacileStudio/gare/apps/api/internal/storage"
)

// recordingSink keeps every event the engine reports, as "kind: message", and the subprocess
// output it streamed, so a test asserts what an operator would have been told without a terminal.
type recordingSink struct {
	mu     sync.Mutex
	events []string
	stdout bytes.Buffer
	stderr bytes.Buffer
}

// kindName labels a kind in a recorded event.
func kindName(kind Kind) string {
	return [...]string{"step", "done", "warn", "detail"}[kind]
}

func (s *recordingSink) Event(_ string, kind Kind, msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, kindName(kind)+": "+msg)
}

func (s *recordingSink) Output(string) (io.Writer, io.Writer) {
	return &s.stdout, &s.stderr
}

// recorded returns a copy of the events reported so far.
func (s *recordingSink) recorded() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.events...)
}

// hasEvent reports whether an event starting with prefix was reported.
func (s *recordingSink) hasEvent(prefix string) bool {
	for _, event := range s.recorded() {
		if strings.HasPrefix(event, prefix) {
			return true
		}
	}
	return false
}

// newTestEngine returns an engine on store reporting to a fresh recording sink.
func newTestEngine(store *storage.Store, settings Settings) (*Engine, *recordingSink) {
	sink := &recordingSink{}
	return New(store, sink, settings), sink
}

func fakeCaddy(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	script := filepath.Join(dir, "caddy")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatalf("failed to write a fake caddy: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

type testServer struct {
	server *httptest.Server
	port   int
}

func newTestServer(t *testing.T, routes map[string]int) testServer {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		status, found := routes[r.URL.Path]
		if !found {
			status = http.StatusNotFound
		}
		w.WriteHeader(status)
	}))
	t.Cleanup(server.Close)
	return testServer{server: server, port: server.Listener.Addr().(*net.TCPAddr).Port}
}

func closedPort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	return port
}
