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
		Name:      "myapp",
		AppType:   "static",
		Port:      8000,
		Domains:   []string{"myapp.local"},
		RepoURL:   "https://github.com/example/myapp",
		Branch:    "main",
		CreatedAt: "2026-09-17T00:00:00Z",
		Health:    &storage.HealthSection{Path: "/health"},
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
	if details.Name != "myapp" || details.Healthcheck != "/health" || len(details.Domains) != 1 || details.Domains[0] != "myapp.local" {
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
		Domains: []string{"human.local"},
		RepoURL: "https://github.com/example/humanapp",
		Branch:  "main",
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
	assertNestedUnderASection(t, output, "Type:   static")
	assertNestedUnderASection(t, output, "Branch: main")
}

// assertNestedUnderASection guards the lipgloss tree API's trap: tree.Child returns its receiver, so
// a section built from that return value appends its children to the root and they render as
// siblings of the section instead of nested under it.
func assertNestedUnderASection(t *testing.T, output, child string) {
	t.Helper()
	for line := range strings.SplitSeq(output, "\n") {
		if !strings.Contains(line, child) {
			continue
		}
		if strings.HasPrefix(line, "\u251c\u2500\u2500") || strings.HasPrefix(line, "\u2514\u2500\u2500") {
			t.Errorf("expected %q to be nested under its section, but it rendered at the root:\n%s", child, output)
		}
		return
	}
	t.Errorf("expected %q in the status output:\n%s", child, output)
}
