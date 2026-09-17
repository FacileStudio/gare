package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManifestEnvGetAndSet(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "manifest.yaml")
	if err := GenerateDefaultManifest("testapp", 8080, manifestPath); err != nil {
		t.Fatalf("failed to generate default manifest: %v", err)
	}

	envs, err := GetManifestEnv(manifestPath)
	if err != nil || len(envs) != 0 {
		t.Fatalf("expected 0 envs, got %d, err: %v", len(envs), err)
	}

	setVars := map[string]string{"PORT": "8080", "DB_URL": "postgres://localhost/db"}
	if err := SetManifestEnv(manifestPath, setVars); err != nil {
		t.Fatalf("SetManifestEnv failed: %v", err)
	}

	envs, err = GetManifestEnv(manifestPath)
	if err != nil || len(envs) != 2 || envs["PORT"] != "8080" {
		t.Fatalf("expected 2 envs with PORT=8080, got %+v, err: %v", envs, err)
	}
}

func TestManifestEnvUpdateAndUnset(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "manifest.yaml")
	if err := GenerateDefaultManifest("testapp", 8080, manifestPath); err != nil {
		t.Fatalf("failed to generate default manifest: %v", err)
	}
	if err := SetManifestEnv(manifestPath, map[string]string{"PORT": "8080", "SECRET": "old"}); err != nil {
		t.Fatalf("initial SetManifestEnv failed: %v", err)
	}

	updateVars := map[string]string{"PORT": "9000", "SECRET": "new", "EXTRA": "val"}
	if err := SetManifestEnv(manifestPath, updateVars); err != nil {
		t.Fatalf("SetManifestEnv update failed: %v", err)
	}

	if err := UnsetManifestEnv(manifestPath, []string{"SECRET", "EXTRA"}); err != nil {
		t.Fatalf("UnsetManifestEnv failed: %v", err)
	}

	envs, err := GetManifestEnv(manifestPath)
	if err != nil || len(envs) != 1 || envs["PORT"] != "9000" {
		t.Fatalf("expected 1 env with PORT=9000, got %+v, err: %v", envs, err)
	}
}

func TestLoadDotEnv(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")
	content := "# Comment\n\nFOO=bar\nexport BAZ=\"spaces\"\nSINGLE='single'\nINVALID\n"
	if err := os.WriteFile(envPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write .env: %v", err)
	}

	vars, err := LoadDotEnv(envPath)
	if err != nil || len(vars) != 3 {
		t.Fatalf("expected 3 vars, got %d, err: %v", len(vars), err)
	}
	if vars["FOO"] != "bar" || vars["BAZ"] != "spaces" || vars["SINGLE"] != "single" {
		t.Errorf("unexpected vars values: %+v", vars)
	}
}

func TestParseEnvAssignments(t *testing.T) {
	args := []string{"KEY1=VAL1", "KEY2=VAL2=EXTRA"}
	vars, err := ParseEnvAssignments(args)
	if err != nil || len(vars) != 2 || vars["KEY2"] != "VAL2=EXTRA" {
		t.Fatalf("ParseEnvAssignments mismatch, got %+v, err: %v", vars, err)
	}

	if _, err := ParseEnvAssignments([]string{"INVALID"}); err == nil {
		t.Errorf("expected error for invalid assignment, got nil")
	}
}
