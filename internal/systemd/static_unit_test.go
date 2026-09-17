package systemd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateStaticUnit(t *testing.T) {
	content, err := GenerateStaticUnit("my-static-site", 8000, "/var/www/site")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(content, "Description=Gare Managed Static App: my-static-site") {
		t.Errorf("missing description: %s", content)
	}
	if !strings.Contains(content, "file-server --listen :8000 --root /var/www/site") {
		t.Errorf("missing file-server command: %s", content)
	}
	if !strings.Contains(content, "WantedBy=default.target") {
		t.Errorf("missing install target: %s", content)
	}
}

func TestWriteStaticUnit(t *testing.T) {
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	os.Setenv("HOME", tempDir)

	if err := WriteStaticUnit("site-test", 8005, "/tmp/site"); err != nil {
		t.Fatalf("failed to write static unit: %v", err)
	}

	expectedPath := filepath.Join(tempDir, ".config", "systemd", "user", "site-test.service")
	data, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("failed to read unit file: %v", err)
	}

	if !strings.Contains(string(data), ":8005") {
		t.Errorf("unit missing port 8005: %s", string(data))
	}
}
