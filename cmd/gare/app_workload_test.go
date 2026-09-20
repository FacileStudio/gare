package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveComposeWorkload(t *testing.T) {
	repoDir := t.TempDir()
	writeRepoComposeFile(t, repoDir, "docker-compose.yml", "services:\n  web:\n    ports:\n      - \"8000:80\"\n")

	composeFile, err := resolveComposeWorkload(repoDir, "", 8000)
	if err != nil || composeFile != "docker-compose.yml" {
		t.Errorf("resolveComposeWorkload: got %q, err %v", composeFile, err)
	}
	if _, err := resolveComposeWorkload(repoDir, "", 8001); err == nil {
		t.Error("expected error when the port is not published by the compose file")
	}
	if _, err := resolveComposeWorkload(repoDir, "", 0); err == nil {
		t.Error("expected error when no port is configured")
	}
}

func TestComposeProjectName(t *testing.T) {
	if got := composeProjectName("My-App"); got != "my-app" {
		t.Errorf("composeProjectName: got %q, want my-app", got)
	}
}

func TestNormalizeCreateOptionsCompose(t *testing.T) {
	repoDir := t.TempDir()
	writeRepoComposeFile(t, repoDir, "compose.yml", "services:\n  web:\n    ports:\n      - \"8700:80\"\n")
	opts := appCreateOptions{appType: "compose", port: 8700}
	resolved, err := normalizeCreateOptions(repoDir, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.composeFile != "compose.yml" || resolved.appType != "compose" {
		t.Errorf("unexpected resolved options: %+v", resolved)
	}
	if resolved.containerPort != 0 {
		t.Errorf("compose workloads must not detect a container port, got %d", resolved.containerPort)
	}
}

func TestNormalizeCreateOptionsMissingComposeFile(t *testing.T) {
	opts := appCreateOptions{appType: "compose", port: 8700}
	if _, err := normalizeCreateOptions(t.TempDir(), opts); err == nil {
		t.Error("expected error when no compose file exists")
	}
}

func writeRepoComposeFile(t *testing.T, repoDir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(repoDir, name), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
