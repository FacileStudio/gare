package manifest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdatePorts(t *testing.T) {
	manifestFile := filepath.Join(t.TempDir(), "manifest.yaml")
	if err := Generate("app", 3000, 8000, manifestFile); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if err := SetEnv(manifestFile, map[string]string{"ENV_VAR": "keep_me"}); err != nil {
		t.Fatalf("SetEnv failed: %v", err)
	}
	if err := UpdatePorts(manifestFile, 8080, 9000); err != nil {
		t.Fatalf("UpdatePorts failed: %v", err)
	}

	content, err := os.ReadFile(manifestFile)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	if !strings.Contains(text, "containerPort: 8080") || !strings.Contains(text, "hostPort: 9000") {
		t.Errorf("expected updated ports in manifest:\n%s", text)
	}
	envs, err := GetEnv(manifestFile)
	if err != nil || envs["ENV_VAR"] != "keep_me" {
		t.Errorf("expected env var preserved, got: %+v, err: %v", envs, err)
	}
}

func TestUpdatePortsUpsertMissingKey(t *testing.T) {
	manifestFile := filepath.Join(t.TempDir(), "manifest.yaml")
	minimal := "apiVersion: v1\nkind: Pod\nspec:\n  containers:\n  - name: test\n    ports:\n    - containerPort: 3000\n"
	if err := os.WriteFile(manifestFile, []byte(minimal), 0644); err != nil {
		t.Fatal(err)
	}

	if err := UpdatePorts(manifestFile, 3000, 8080); err != nil {
		t.Fatalf("UpdatePorts failed: %v", err)
	}
	content, err := os.ReadFile(manifestFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "hostPort: 8080") {
		t.Errorf("expected upserted hostPort in manifest:\n%s", string(content))
	}
}

func TestUpdatePortsValidation(t *testing.T) {
	manifestFile := filepath.Join(t.TempDir(), "manifest.yaml")
	if err := Generate("app", 3000, 8000, manifestFile); err != nil {
		t.Fatal(err)
	}
	if err := UpdatePorts(manifestFile, 3000, 70000); err == nil {
		t.Error("expected error for hostPort > 65535, got nil")
	}
	if err := UpdatePorts(manifestFile, 70000, 8000); err == nil {
		t.Error("expected error for containerPort > 65535, got nil")
	}
}
