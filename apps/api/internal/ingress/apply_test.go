package ingress

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/FacileStudio/gare/apps/api/internal/storage"
)

// probingIngress returns an ingress whose reload always succeeds and whose probe is the real one,
// so a test exercises verification without a Caddy on the machine.
func probingIngress(t *testing.T) *Ingress {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("GARE_CADDYFILE", filepath.Join(dir, "Caddyfile"))
	return &Ingress{ConfDir: dir, Reload: func(context.Context) error { return nil }, Probe: ProbeTarget}
}

// closedPort returns a loopback port nothing listens on.
func closedPort(t *testing.T) int {
	t.Helper()
	port, listener := openPort(t)
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	return port
}

// answeringPort serves 200 on a loopback port for the life of the test.
func answeringPort(t *testing.T) int {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	t.Cleanup(server.Close)
	return server.Listener.Addr().(*net.TCPAddr).Port
}

func TestTargets(t *testing.T) {
	defaults := Targets([]string{"app.example.com"})
	want := []Target{{Host: "app.example.com", Port: 80}, {Host: "app.example.com", Port: 443, Secure: true}}
	if !slices.Equal(defaults, want) {
		t.Fatalf("expected the standard ingress ports, got: %v", defaults)
	}

	explicit := Targets([]string{"app.example.com:8443", "secure.example.com:443", "plain.example.com:80"})
	want = []Target{
		{Host: "app.example.com", Port: 8443},
		{Host: "secure.example.com", Port: 443, Secure: true},
		{Host: "plain.example.com", Port: 80},
	}
	if !slices.Equal(explicit, want) {
		t.Fatalf("expected every domain to keep its own hostname and port, got: %v", explicit)
	}
}

func TestTargetsKeepsTheStandardPairBesideAnExplicitPort(t *testing.T) {
	mixed := Targets([]string{"plain.example.com", "custom.example.com:8443"})
	want := []Target{
		{Host: "plain.example.com", Port: 80},
		{Host: "plain.example.com", Port: 443, Secure: true},
		{Host: "custom.example.com", Port: 8443},
	}
	if !slices.Equal(mixed, want) {
		t.Fatalf("expected a hostname on the standard ports beside the explicit one, got: %v", mixed)
	}
}

func TestApplySkipsTheProbeForAppsWithoutDomains(t *testing.T) {
	if err := probingIngress(t).Apply(context.Background(), &storage.AppConfig{Name: "bare", Port: 8000}); err != nil {
		t.Fatalf("expected an app without domains to skip ingress verification, got: %v", err)
	}
}

// TestApplyNamesOnlyTheSilentHostname pins a partial failure: the report names the hostname nothing
// answered for and leaves the answering one out, so the operator does not chase the domain that works.
func TestApplyNamesOnlyTheSilentHostname(t *testing.T) {
	silent := closedPort(t)
	cfg := &storage.AppConfig{Name: "partial", Port: 8000, Domains: []string{
		fmt.Sprintf("served.example.com:%d", answeringPort(t)),
		fmt.Sprintf("silent.example.com:%d", silent),
	}}

	err := probingIngress(t).Apply(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected the silent hostname to fail ingress verification")
	}
	for _, needle := range []string{"partial", "silent.example.com", strconv.Itoa(silent)} {
		if !strings.Contains(err.Error(), needle) {
			t.Fatalf("expected the error to mention %q, got: %v", needle, err)
		}
	}
	if strings.Contains(err.Error(), "served.example.com") {
		t.Fatalf("expected the error to leave the answering hostname out, got: %v", err)
	}
}
