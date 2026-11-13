package engine

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/FacileStudio/gare/apps/api/internal/health"
)

func TestVerifyHealthProbes(t *testing.T) {
	api := newTestServer(t, map[string]int{"/health": http.StatusOK})
	admin := newTestServer(t, map[string]int{"/": http.StatusOK})

	probes := []health.Probe{
		{Name: "api", Port: api.port, Path: "/health"},
		{Name: "admin", Port: admin.port, Path: "/"},
	}
	if err := verifyHealthProbes(context.Background(), reporter{sink: &recordingSink{}}, probes, "smoke"); err != nil {
		t.Errorf("expected both probes to pass: %v", err)
	}
}

func TestVerifyHealthProbesFailsOnUnhealthyPath(t *testing.T) {
	api := newTestServer(t, map[string]int{"/health": http.StatusTeapot})
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()

	err := verifyHealthProbes(ctx, reporter{sink: &recordingSink{}}, []health.Probe{{Name: "api", Port: api.port, Path: "/health"}}, "smoke")
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

	err := verifyHealthProbes(ctx, reporter{sink: &recordingSink{}}, []health.Probe{{Port: closed}}, "smoke")
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
		if got := ProbeTarget(health.Probe{Port: 8100, Path: tc.path}); got != tc.want {
			t.Errorf("probeTarget(%q): got %q, want %q", tc.path, got, tc.want)
		}
	}
}
