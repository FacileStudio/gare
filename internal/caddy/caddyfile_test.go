package caddy

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureCaddyfileCreatesDefaultWhenMissing(t *testing.T) {
	tempDir := t.TempDir()
	caddyfilePath := filepath.Join(tempDir, "Caddyfile")
	confDir := filepath.Join(tempDir, "conf.d")

	if err := EnsureCaddyfilePaths(caddyfilePath, confDir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(caddyfilePath)
	if err != nil {
		t.Fatalf("failed to read created Caddyfile: %v", err)
	}

	expected := "import " + confDir + "/*.caddy\n"
	if string(content) != expected {
		t.Errorf("got %q, want %q", string(content), expected)
	}

	if _, err := os.Stat(confDir); os.IsNotExist(err) {
		t.Errorf("expected snippet directory %s to exist", confDir)
	}
}

func TestEnsureCaddyfileAppendsToExisting(t *testing.T) {
	tempDir := t.TempDir()
	caddyfilePath := filepath.Join(tempDir, "Caddyfile")
	confDir := filepath.Join(tempDir, "conf.d")

	initial := "example.org {\n\treverse_proxy localhost:3000\n}\n"
	if err := os.WriteFile(caddyfilePath, []byte(initial), 0644); err != nil {
		t.Fatalf("failed to write initial Caddyfile: %v", err)
	}

	if err := EnsureCaddyfilePaths(caddyfilePath, confDir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(caddyfilePath)
	if err != nil {
		t.Fatalf("failed to read updated Caddyfile: %v", err)
	}

	expected := strings.TrimSpace(initial) + "\n\nimport " + confDir + "/*.caddy\n"
	if string(content) != expected {
		t.Errorf("got %q, want %q", string(content), expected)
	}
}

func TestEnsureCaddyfileIdempotentWhenImportPresent(t *testing.T) {
	tempDir := t.TempDir()
	caddyfilePath := filepath.Join(tempDir, "Caddyfile")
	confDir := filepath.Join(tempDir, "conf.d")

	initial := "example.org {\n\treverse_proxy localhost:3000\n}\nimport " + confDir + "/*.caddy\n"
	if err := os.WriteFile(caddyfilePath, []byte(initial), 0644); err != nil {
		t.Fatalf("failed to write initial Caddyfile: %v", err)
	}

	if err := EnsureCaddyfilePaths(caddyfilePath, confDir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(caddyfilePath)
	if err != nil {
		t.Fatalf("failed to read Caddyfile: %v", err)
	}

	if string(content) != initial {
		t.Errorf("got %q, want %q (content was modified unexpectedly)", string(content), initial)
	}
}

func TestEnsureCaddyfileWithEnvOverrides(t *testing.T) {
	tempDir := t.TempDir()
	caddyfilePath := filepath.Join(tempDir, "custom-caddyfile")
	confDir := filepath.Join(tempDir, "custom-conf.d")

	t.Setenv("GARE_CADDYFILE", caddyfilePath)
	t.Setenv("GARE_CADDY_CONF_DIR", confDir)

	if err := EnsureCaddyfile(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(caddyfilePath)
	if err != nil {
		t.Fatalf("failed to read created custom Caddyfile: %v", err)
	}

	expected := "import " + confDir + "/*.caddy\n"
	if string(content) != expected {
		t.Errorf("got %q, want %q", string(content), expected)
	}
}

func TestReloadOfflineCaddy(t *testing.T) {
	tempDir := t.TempDir()
	caddyfilePath := filepath.Join(tempDir, "Caddyfile")
	confDir := filepath.Join(tempDir, "conf.d")

	t.Setenv("GARE_CADDYFILE", caddyfilePath)
	t.Setenv("GARE_CADDY_CONF_DIR", confDir)

	ctx := context.Background()
	if err := Reload(ctx); err != nil {
		t.Fatalf("expected Reload to succeed when Caddy is offline, got: %v", err)
	}
}
