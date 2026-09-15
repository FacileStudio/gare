package caddy

import (
	"os"
	"path/filepath"
	"strings"
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
