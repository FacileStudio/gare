package caddy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateStaticServerConfig(t *testing.T) {
	content, err := GenerateStaticServerConfig("/srv", 80)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		":80 {",
		`root * "/srv"`,
		"try_files {path} /index.html",
		"file_server",
	}
	for _, snippet := range expected {
		if !strings.Contains(content, snippet) {
			t.Errorf("expected config to contain %q, got:\n%s", snippet, content)
		}
	}
}

func TestGenerateStaticServerConfigKeepsSPAFallback(t *testing.T) {
	content, err := GenerateStaticServerConfig("/srv", 80)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	before, _, found := strings.Cut(content, "try_files {path} /index.html")
	if !found {
		t.Fatalf("static sites serve deep links through the SPA fallback, got:\n%s", content)
	}
	if strings.Contains(before, "file_server") {
		t.Errorf("try_files must be applied before file_server, got:\n%s", content)
	}
}

func TestGenerateStaticServerConfigRejectsBadPort(t *testing.T) {
	for _, port := range []int{0, -1, 70000} {
		if _, err := GenerateStaticServerConfig("/srv", port); err == nil {
			t.Errorf("expected an error for port %d", port)
		}
	}
}

func TestWriteStaticServerConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Caddyfile")
	if err := WriteStaticServerConfig(path, "/srv", 80); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read the Caddyfile: %v", err)
	}
	if !strings.Contains(string(content), "try_files {path} /index.html") {
		t.Errorf("written Caddyfile lost the SPA fallback:\n%s", string(content))
	}
}
