package manifest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderEnvInjectsIntoAGeneratedManifest(t *testing.T) {
	manifestPath := filepath.Join(t.TempDir(), "manifest.yaml")
	if err := Generate("testapp", 8080, 8080, manifestPath); err != nil {
		t.Fatalf("failed to generate default manifest: %v", err)
	}
	if envs, err := GetEnv(manifestPath); err != nil || len(envs) != 0 {
		t.Fatalf("expected 0 envs, got %v, err: %v", envs, err)
	}
	setVars := map[string]string{"PORT": "8080", "DEBUG": "true", "DB_URL": "postgres://localhost/db"}
	if err := RenderEnv(manifestPath, "", setVars); err != nil {
		t.Fatalf("RenderEnv failed: %v", err)
	}
	envs, err := GetEnv(manifestPath)
	if err != nil || len(envs) != 3 || envs["PORT"] != "8080" || envs["DEBUG"] != "true" {
		t.Fatalf("expected the three variables, got %+v, err: %v", envs, err)
	}
	data, _ := os.ReadFile(manifestPath)
	if !strings.Contains(string(data), `value: "true"`) || !strings.Contains(string(data), `value: "8080"`) {
		t.Errorf("expected scalar-looking values to stay strings, got:\n%s", data)
	}
	if err := RenderEnv(manifestPath, "", map[string]string{"PORT": "9000"}); err != nil {
		t.Fatalf("second RenderEnv failed: %v", err)
	}
	if envs, _ := GetEnv(manifestPath); len(envs) != 1 || envs["PORT"] != "9000" {
		t.Fatalf("expected a re-render to drop variables the env file no longer holds, got %+v", envs)
	}
}

func TestRenderEnvKeepsTheRepositoryEntriesAndLetsTheEnvFileWin(t *testing.T) {
	dir := t.TempDir()
	repoManifest := filepath.Join(dir, "repo.yaml")
	content := "apiVersion: v1\nkind: Pod\nmetadata:\n  name: app\nspec:\n  containers:\n  - name: app\n" +
		"    image: localhost/app:latest\n    env:\n    - name: FROM_REPO\n      value: kept\n" +
		"    - name: CLASH\n      value: repo\n    - name: POD_IP\n      valueFrom:\n" +
		"        fieldRef:\n          fieldPath: status.podIP\n"
	if err := os.WriteFile(repoManifest, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(dir, "manifest.yaml")
	if err := os.WriteFile(manifestPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	if err := RenderEnv(manifestPath, repoManifest, map[string]string{"CLASH": "file", "SECRET": "s"}); err != nil {
		t.Fatalf("RenderEnv failed: %v", err)
	}
	envs, err := GetEnv(manifestPath)
	if err != nil || envs["FROM_REPO"] != "kept" || envs["CLASH"] != "file" || envs["SECRET"] != "s" {
		t.Fatalf("expected the repo entry kept and the env file to win, got %+v, err: %v", envs, err)
	}
	if data, _ := os.ReadFile(manifestPath); !strings.Contains(string(data), "fieldPath: status.podIP") {
		t.Errorf("expected the repository's valueFrom entry to survive, got:\n%s", data)
	}
}
