package main

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/FacileStudio/gare/internal/storage"
)

func TestVerifyHealthProbes(t *testing.T) {
	api := newTestServer(t, map[string]int{"/health": http.StatusOK})
	admin := newTestServer(t, map[string]int{"/": http.StatusOK})

	probes := []storage.HealthProbe{
		{Name: "api", Port: api.port, Path: "/health"},
		{Name: "admin", Port: admin.port, Path: "/"},
	}
	if err := verifyHealthProbes(context.Background(), probes); err != nil {
		t.Errorf("expected both probes to pass: %v", err)
	}
}

func TestVerifyHealthProbesFailsOnUnhealthyPath(t *testing.T) {
	api := newTestServer(t, map[string]int{"/health": http.StatusTeapot})
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()

	err := verifyHealthProbes(ctx, []storage.HealthProbe{{Name: "api", Port: api.port, Path: "/health"}})
	if err == nil {
		t.Fatal("expected a failing probe to error")
	}
	if !strings.Contains(err.Error(), "api") {
		t.Errorf("error must name the failing probe, got %v", err)
	}
}

func TestVerifyHealthProbesFailsOnClosedPort(t *testing.T) {
	closed := closedPort(t)
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()

	err := verifyHealthProbes(ctx, []storage.HealthProbe{{Port: closed}})
	if err == nil {
		t.Fatal("expected an unreachable probe to error")
	}
	if !strings.Contains(err.Error(), "port") {
		t.Errorf("error must name the probe by port, got %v", err)
	}
}

func TestProbeTarget(t *testing.T) {
	cases := []struct {
		path string
		want string
	}{
		{"", "http://127.0.0.1:8100/"},
		{"/", "http://127.0.0.1:8100/"},
		{"health", "http://127.0.0.1:8100/health"},
		{"/readyz", "http://127.0.0.1:8100/readyz"},
	}
	for _, tc := range cases {
		if got := probeTarget(storage.HealthProbe{Port: 8100, Path: tc.path}); got != tc.want {
			t.Errorf("probeTarget(%q): got %q, want %q", tc.path, got, tc.want)
		}
	}
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
