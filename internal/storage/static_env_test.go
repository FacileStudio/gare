package storage

import (
	"testing"
)

func TestStaticEnvSet(t *testing.T) {
	appDir := t.TempDir()
	envs, err := GetStaticEnv(appDir)
	if err != nil {
		t.Fatalf("unexpected error on missing env file: %v", err)
	}
	if len(envs) != 0 {
		t.Errorf("expected empty map, got %+v", envs)
	}
	setVars := map[string]string{"PORT": "8080", "API": "https://example.com"}
	if err := SetStaticEnv(appDir, setVars); err != nil {
		t.Fatalf("failed to set static env: %v", err)
	}
	envs, err = GetStaticEnv(appDir)
	if err != nil || envs["PORT"] != "8080" || envs["API"] != "https://example.com" {
		t.Fatalf("unexpected env map: %+v, err: %v", envs, err)
	}
}

func TestStaticEnvUnset(t *testing.T) {
	appDir := t.TempDir()
	setVars := map[string]string{"PORT": "8080", "API": "https://example.com"}
	if err := SetStaticEnv(appDir, setVars); err != nil {
		t.Fatal(err)
	}
	if err := UnsetStaticEnv(appDir, []string{"PORT"}); err != nil {
		t.Fatalf("failed to unset key: %v", err)
	}
	envs, err := GetStaticEnv(appDir)
	if err != nil || envs["API"] != "https://example.com" {
		t.Fatalf("unexpected API env: %+v", envs)
	}
	if _, ok := envs["PORT"]; ok {
		t.Errorf("expected PORT to be deleted")
	}
}
