package engine

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/FacileStudio/gare/apps/api/internal/storage"
)

func TestActivateIngress(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("GARE_CADDY_CONF_DIR", filepath.Join(tmp, "caddy", "conf.d"))
	t.Setenv("GARE_CADDYFILE", filepath.Join(tmp, "caddy", "Caddyfile"))
	fakeCaddy(t)
	ctx := context.Background()
	eng, _ := newTestEngine(storage.Open(tmp), Settings{})

	cfg := &storage.AppConfig{Name: "svc", Port: 8000, Domains: []string{fmt.Sprintf("svc.example.com:%d", closedPort(t))}}
	if err := eng.activateIngress(ctx, cfg); err == nil {
		t.Fatal("expected activateIngress to fail when nothing answers for the domain port")
	}

	ingress := newTestServer(t, map[string]int{"/": http.StatusOK})
	cfg.Domains = []string{fmt.Sprintf("svc.example.com:%d", ingress.port)}
	if err := eng.activateIngress(ctx, cfg); err != nil {
		t.Fatalf("expected activateIngress to pass with an answering ingress, got: %v", err)
	}

	cfg.Domains = nil
	if err := eng.activateIngress(ctx, cfg); err != nil {
		t.Fatalf("expected a domainless app to skip the ingress probe, got: %v", err)
	}
}
