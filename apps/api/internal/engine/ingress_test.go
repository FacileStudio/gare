package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FacileStudio/gare/apps/api/internal/storage"
)

func TestSyncAppIngress(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	confDir := filepath.Join(tmpDir, "caddy", "conf.d")
	t.Setenv("GARE_CADDY_CONF_DIR", confDir)
	caddyfilePath := filepath.Join(tmpDir, "caddy", "Caddyfile")
	t.Setenv("GARE_CADDYFILE", caddyfilePath)

	cfg := &storage.AppConfig{Name: "capp", Port: 8080, Domains: []string{"c1.example.com", "c2.example.com"}}
	if err := syncAppIngress(cfg); err != nil {
		t.Fatalf("syncAppIngress failed: %v", err)
	}

	snippetPath := filepath.Join(confDir, "capp.caddy")
	data, err := os.ReadFile(snippetPath)
	if err != nil || !strings.Contains(string(data), "c1.example.com, c2.example.com") {
		t.Fatalf("snippet missing expected content: %v, content: %s", err, string(data))
	}

	cfg.Domains = nil
	if err := syncAppIngress(cfg); err != nil {
		t.Fatalf("syncAppIngress remove failed: %v", err)
	}
	if _, err := os.Stat(snippetPath); !os.IsNotExist(err) {
		t.Fatal("expected snippet to be removed when no domains exist")
	}
}
