package caddy

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestGenerateSnippet(t *testing.T) {
	snippet, err := GenerateSnippet("example.com", 8080)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "example.com {\n\treverse_proxy localhost:8080\n}\n"
	if snippet != expected {
		t.Errorf("got %q, want %q", snippet, expected)
	}
}

func TestWriteAndRemoveSnippet(t *testing.T) {
	tempDir := t.TempDir()
	name := "testapp"
	domain := "test.example.com"
	port := 8001

	if err := WriteSnippet(tempDir, name, domain, port); err != nil {
		t.Fatalf("unexpected error writing snippet: %v", err)
	}

	path := GetSnippetPath(tempDir, name)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read snippet file: %v", err)
	}

	if !strings.Contains(string(content), "reverse_proxy localhost:8001") {
		t.Errorf("unexpected content: %s", string(content))
	}

	if err := RemoveSnippet(tempDir, name); err != nil {
		t.Fatalf("unexpected error removing snippet: %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("expected file to be deleted, but it still exists")
	}
}

func TestGetSnippetPath(t *testing.T) {
	p := GetSnippetPath("", "demo")
	expected := filepath.Join(DefaultConfDir, "demo.caddy")
	if p != expected {
		t.Errorf("got %q, want %q", p, expected)
	}
}

func TestGenerateStaticSnippet(t *testing.T) {
	snippet, err := GenerateStaticSnippet("example.com", "/var/www/html")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "example.com {\n\troot * \"/var/www/html\"\n\tfile_server\n\ttry_files {path} /index.html\n}\n"
	if snippet != expected {
		t.Errorf("got %q, want %q", snippet, expected)
	}
}

func TestWriteStaticSnippet(t *testing.T) {
	tempDir := t.TempDir()
	name := "staticsite"
	domain := "static.example.com"
	rootDir := "/var/www/site"

	if err := WriteStaticSnippet(tempDir, name, domain, rootDir); err != nil {
		t.Fatalf("unexpected error writing static snippet: %v", err)
	}

	path := GetSnippetPath(tempDir, name)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read static snippet file: %v", err)
	}

	expected := "static.example.com {\n\troot * \"/var/www/site\"\n\tfile_server\n\ttry_files {path} /index.html\n}\n"
	if string(content) != expected {
		t.Errorf("got %q, want %q", string(content), expected)
	}
}

func TestEnsureCaddyfileAtPath(t *testing.T) {
	tempDir := t.TempDir()
	caddyfilePath := filepath.Join(tempDir, "Caddyfile")

	if err := EnsureCaddyfileAtPath(caddyfilePath); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(caddyfilePath)
	if err != nil {
		t.Fatalf("failed to read created Caddyfile: %v", err)
	}

	if string(content) != defaultCaddyfileContent {
		t.Errorf("got %q, want %q", string(content), defaultCaddyfileContent)
	}

	if err := EnsureCaddyfileAtPath(caddyfilePath); err != nil {
		t.Fatalf("unexpected error on second call: %v", err)
	}
}

func TestIsSpaceError(t *testing.T) {
	if !errors.Is(syscall.ENOSPC, syscall.ENOSPC) {
		t.Error("expected isSpaceError to return true for ENOSPC")
	}
	if errors.Is(syscall.EDQUOT, syscall.ENOSPC) {
		t.Error("expected isSpaceError to return false for EDQUOT")
	}
	if errors.Is(errors.New("some other error"), syscall.ENOSPC) {
		t.Error("expected isSpaceError to return false for unrelated errors")
	}
}

func TestIsQuotaExceeded(t *testing.T) {
	if !errors.Is(syscall.EDQUOT, syscall.EDQUOT) {
		t.Error("expected isQuotaExceeded to return true for EDQUOT")
	}
	if errors.Is(syscall.ENOSPC, syscall.EDQUOT) {
		t.Error("expected isQuotaExceeded to return false for ENOSPC")
	}
	if errors.Is(errors.New("some other error"), syscall.EDQUOT) {
		t.Error("expected isQuotaExceeded to return false for unrelated errors")
	}
}
