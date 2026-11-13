package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FacileStudio/gare/apps/api/internal/storage"
)

func TestActivateIngressWritesAndRemovesTheSnippet(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	confDir := filepath.Join(tmpDir, "caddy", "conf.d")
	t.Setenv("GARE_CADDY_CONF_DIR", confDir)
	caddyfilePath := filepath.Join(tmpDir, "caddy", "Caddyfile")
	t.Setenv("GARE_CADDYFILE", caddyfilePath)
	fakeCaddy(t)
	eng, _ := newTestEngine(storage.Open(tmpDir), Settings{})
	answering := newTestServer(t, map[string]int{"/": 200})

	cfg := &storage.AppConfig{Name: "capp", Port: 8080, Domains: []string{
		fmt.Sprintf("c1.example.com:%d", answering.port), fmt.Sprintf("c2.example.com:%d", answering.port),
	}}
	if err := eng.activateIngress(context.Background(), cfg); err != nil {
		t.Fatalf("activateIngress failed: %v", err)
	}

	snippetPath := filepath.Join(confDir, "capp.caddy")
	data, err := os.ReadFile(snippetPath)
	if err != nil || !strings.Contains(string(data), fmt.Sprintf("c1.example.com:%d, c2.example.com:%d", answering.port, answering.port)) {
		t.Fatalf("snippet missing expected content: %v, content: %s", err, string(data))
	}

	cfg.Domains = nil
	if err := eng.activateIngress(context.Background(), cfg); err != nil {
		t.Fatalf("activateIngress remove failed: %v", err)
	}
	if _, err := os.Stat(snippetPath); !os.IsNotExist(err) {
		t.Fatal("expected snippet to be removed when no domains exist")
	}
}
