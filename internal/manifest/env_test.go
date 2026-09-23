package manifest

import (
	"path/filepath"
	"testing"
)

func TestEnvGetAndSet(t *testing.T) {
	manifestPath := filepath.Join(t.TempDir(), "manifest.yaml")
	if err := Generate("testapp", 8080, 8080, manifestPath); err != nil {
		t.Fatalf("failed to generate default manifest: %v", err)
	}

	envs, err := GetEnv(manifestPath)
	if err != nil || len(envs) != 0 {
		t.Fatalf("expected 0 envs, got %d, err: %v", len(envs), err)
	}

	setVars := map[string]string{"PORT": "8080", "DB_URL": "postgres://localhost/db"}
	if err := SetEnv(manifestPath, setVars); err != nil {
		t.Fatalf("SetEnv failed: %v", err)
	}

	envs, err = GetEnv(manifestPath)
	if err != nil || len(envs) != 2 || envs["PORT"] != "8080" {
		t.Fatalf("expected 2 envs with PORT=8080, got %+v, err: %v", envs, err)
	}
}

func TestEnvUpdateAndUnset(t *testing.T) {
	manifestPath := filepath.Join(t.TempDir(), "manifest.yaml")
	if err := Generate("testapp", 8080, 8080, manifestPath); err != nil {
		t.Fatalf("failed to generate default manifest: %v", err)
	}
	if err := SetEnv(manifestPath, map[string]string{"PORT": "8080", "SECRET": "old"}); err != nil {
		t.Fatalf("initial SetEnv failed: %v", err)
	}

	updateVars := map[string]string{"PORT": "9000", "SECRET": "new", "EXTRA": "val"}
	if err := SetEnv(manifestPath, updateVars); err != nil {
		t.Fatalf("SetEnv update failed: %v", err)
	}

	if err := UnsetEnv(manifestPath, []string{"SECRET", "EXTRA"}); err != nil {
		t.Fatalf("UnsetEnv failed: %v", err)
	}

	envs, err := GetEnv(manifestPath)
	if err != nil || len(envs) != 1 || envs["PORT"] != "9000" {
		t.Fatalf("expected 1 env with PORT=9000, got %+v, err: %v", envs, err)
	}
}
