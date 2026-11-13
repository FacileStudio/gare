package storage

import (
	"os"
	"testing"
)

func TestSetAppEnvPreservesOtherValues(t *testing.T) {
	appDir := t.TempDir()
	values := map[string]string{"HASH": "ab #cd", "QUOTED": `"q"`, "LEAD": " lead", "MULTI": "l1\nl2"}
	if err := SetAppEnv(appDir, values); err != nil {
		t.Fatalf("SetAppEnv: %v", err)
	}
	if err := SetAppEnv(appDir, map[string]string{"OTHER": "1"}); err != nil {
		t.Fatalf("second SetAppEnv: %v", err)
	}
	if err := UnsetAppEnv(appDir, []string{"OTHER"}); err != nil {
		t.Fatalf("UnsetAppEnv: %v", err)
	}
	got, err := GetAppEnv(appDir)
	if err != nil {
		t.Fatalf("GetAppEnv: %v", err)
	}
	for k, v := range values {
		if got[k] != v {
			t.Errorf("%s: set %q, read back %q", k, v, got[k])
		}
	}
}

func TestAppEnvFileIsOwnerOnly(t *testing.T) {
	appDir := t.TempDir()
	if err := EnsureAppEnvFile(appDir); err != nil {
		t.Fatalf("EnsureAppEnvFile: %v", err)
	}
	if err := SetAppEnv(appDir, map[string]string{"K": "v"}); err != nil {
		t.Fatalf("SetAppEnv: %v", err)
	}
	info, err := os.Stat(GetAppEnvPath(appDir))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("env file mode %v, want 0600", info.Mode().Perm())
	}
}

func TestAppEnvSet(t *testing.T) {
	appDir := t.TempDir()
	envs, err := GetAppEnv(appDir)
	if err != nil {
		t.Fatalf("unexpected error on missing env file: %v", err)
	}
	if len(envs) != 0 {
		t.Errorf("expected empty map, got %+v", envs)
	}
	setVars := map[string]string{"PORT": "8080", "API": "https://example.com"}
	if err := SetAppEnv(appDir, setVars); err != nil {
		t.Fatalf("failed to set app env: %v", err)
	}
	envs, err = GetAppEnv(appDir)
	if err != nil || envs["PORT"] != "8080" || envs["API"] != "https://example.com" {
		t.Fatalf("unexpected env map: %+v, err: %v", envs, err)
	}
}

func TestAppEnvUnset(t *testing.T) {
	appDir := t.TempDir()
	setVars := map[string]string{"PORT": "8080", "API": "https://example.com"}
	if err := SetAppEnv(appDir, setVars); err != nil {
		t.Fatal(err)
	}
	if err := UnsetAppEnv(appDir, []string{"PORT"}); err != nil {
		t.Fatalf("failed to unset key: %v", err)
	}
	envs, err := GetAppEnv(appDir)
	if err != nil || envs["API"] != "https://example.com" {
		t.Fatalf("unexpected API env: %+v", envs)
	}
	if _, ok := envs["PORT"]; ok {
		t.Errorf("expected PORT to be deleted")
	}
}
