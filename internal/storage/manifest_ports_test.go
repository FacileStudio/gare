package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateManifestPorts(t *testing.T) {
	tmpDir := t.TempDir()
	manifestFile := filepath.Join(tmpDir, "manifest.yaml")
	if err := GenerateDefaultManifest("app", 3000, 8000, manifestFile); err != nil {
		t.Fatalf("GenerateDefaultManifest failed: %v", err)
	}
	if err := SetManifestEnv(manifestFile, map[string]string{"ENV_VAR": "keep_me"}); err != nil {
		t.Fatalf("SetManifestEnv failed: %v", err)
	}
	if err := UpdateManifestPorts(manifestFile, 8080, 9000); err != nil {
		t.Fatalf("UpdateManifestPorts failed: %v", err)
	}

	content, err := os.ReadFile(manifestFile)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	if !strings.Contains(text, "containerPort: 8080") || !strings.Contains(text, "hostPort: 9000") {
		t.Errorf("expected updated ports in manifest:\n%s", text)
	}
	envs, err := GetManifestEnv(manifestFile)
	if err != nil || envs["ENV_VAR"] != "keep_me" {
		t.Errorf("expected env var preserved, got: %+v, err: %v", envs, err)
	}
}

func TestUpdateManifestPortsUpsertMissingKey(t *testing.T) {
	tmpDir := t.TempDir()
	manifestFile := filepath.Join(tmpDir, "manifest.yaml")
	minimal := "apiVersion: v1\nkind: Pod\nspec:\n  containers:\n  - name: test\n    ports:\n    - containerPort: 3000\n"
	if err := os.WriteFile(manifestFile, []byte(minimal), 0644); err != nil {
		t.Fatal(err)
	}

	if err := UpdateManifestPorts(manifestFile, 3000, 8080); err != nil {
		t.Fatalf("UpdateManifestPorts failed: %v", err)
	}
	content, err := os.ReadFile(manifestFile)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	if !strings.Contains(text, "hostPort: 8080") {
		t.Errorf("expected upserted hostPort in manifest:\n%s", text)
	}
}

func TestUpdateManifestPortsValidation(t *testing.T) {
	tmpDir := t.TempDir()
	manifestFile := filepath.Join(tmpDir, "manifest.yaml")
	if err := GenerateDefaultManifest("app", 3000, 8000, manifestFile); err != nil {
		t.Fatal(err)
	}
	if err := UpdateManifestPorts(manifestFile, 3000, 70000); err == nil {
		t.Error("expected error for hostPort > 65535, got nil")
	}
	if err := UpdateManifestPorts(manifestFile, 70000, 8000); err == nil {
		t.Error("expected error for containerPort > 65535, got nil")
	}
}
