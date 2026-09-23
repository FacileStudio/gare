package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/FacileStudio/gare/internal/health"
)

func TestVerifyHealthProbes(t *testing.T) {
	api := newTestServer(t, map[string]int{"/health": http.StatusOK})
	admin := newTestServer(t, map[string]int{"/": http.StatusOK})

	probes := []health.Probe{
		{Name: "api", Port: api.port, Path: "/health"},
		{Name: "admin", Port: admin.port, Path: "/"},
	}
	if err := verifyHealthProbes(context.Background(), probes, "smoke"); err != nil {
		t.Errorf("expected both probes to pass: %v", err)
	}
}

func TestVerifyHealthProbesFailsOnUnhealthyPath(t *testing.T) {
	api := newTestServer(t, map[string]int{"/health": http.StatusTeapot})
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()

	err := verifyHealthProbes(ctx, []health.Probe{{Name: "api", Port: api.port, Path: "/health"}}, "smoke")
	if err == nil {
		t.Fatal("expected a failing probe to error")
	}
	if !strings.Contains(err.Error(), "api") {
		t.Errorf("error must name the failing probe, got %v", err)
	}
	if !strings.Contains(err.Error(), "gare logs smoke") {
		t.Errorf("error must point at the logs, got %v", err)
	}
}

func TestVerifyHealthProbesFailsOnClosedPort(t *testing.T) {
	closed := closedPort(t)
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()

	err := verifyHealthProbes(ctx, []health.Probe{{Port: closed}}, "smoke")
	if err == nil {
		t.Fatal("expected an unreachable probe to error")
	}
	if !strings.Contains(err.Error(), "port") {
		t.Errorf("error must name the probe by port, got %v", err)
	}
}

func TestProbeSummaryCapsLongLists(t *testing.T) {
	probes := make([]health.Probe, 0, 6)
	for i := range 6 {
		probes = append(probes, health.Probe{Name: fmt.Sprintf("svc%d", i), Port: 8100 + i})
	}
	summary := probeSummary(probes)
	if !strings.Contains(summary, "and 2 more") {
		t.Errorf("summary must cap the list, got %q", summary)
	}
	if strings.Contains(summary, "svc5") {
		t.Errorf("summary must omit the trailing labels, got %q", summary)
	}
	if short := probeSummary(probes[:2]); short != "svc0, svc1" {
		t.Errorf("short summary: got %q, want %q", short, "svc0, svc1")
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
		if got := probeTarget(health.Probe{Port: 8100, Path: tc.path}); got != tc.want {
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
