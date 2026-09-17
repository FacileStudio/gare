package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FacileStudio/gare/internal/storage"
)

func TestStatusCmdJSON(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	appDir := filepath.Join(tmpDir, ".local", "share", "gare", "apps", "myapp")
	cfg := &storage.AppConfig{
		Name:        "myapp",
		AppType:     "static",
		Port:        8000,
		Domain:      "myapp.local",
		RepoURL:     "https://github.com/example/myapp",
		Branch:      "main",
		CreatedAt:   "2026-09-17T00:00:00Z",
		Healthcheck: "/health",
	}
	if err := storage.SaveConfig(appDir, cfg); err != nil {
		t.Fatal(err)
	}

	cmd := NewStatusCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"myapp", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("status --json failed: %v", err)
	}

	var details AppStatusDetails
	if err := json.Unmarshal(buf.Bytes(), &details); err != nil {
		t.Fatalf("invalid json: %v\noutput: %s", err, buf.String())
	}
	if details.Name != "myapp" || details.Healthcheck != "/health" {
		t.Errorf("status details mismatch: %+v", details)
	}
}

func TestStatusCmdHuman(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	appDir := filepath.Join(tmpDir, ".local", "share", "gare", "apps", "humanapp")
	cfg := &storage.AppConfig{
		Name:    "humanapp",
		AppType: "static",
		Domain:  "human.local",
	}
	if err := storage.SaveConfig(appDir, cfg); err != nil {
		t.Fatal(err)
	}

	cmd := NewStatusCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"humanapp"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("status failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "humanapp") || !strings.Contains(output, "Workload") {
		t.Errorf("expected human output to contain app name and Workload, got: %s", output)
	}
}
