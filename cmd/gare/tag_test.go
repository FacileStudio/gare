package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FacileStudio/gare/internal/storage"
)

func setupTagTestApp(t *testing.T, tmpDir, name string, tags []string) *storage.AppConfig {
	t.Helper()
	appDir := filepath.Join(tmpDir, ".local", "share", "gare", "apps", name)
	cfg := &storage.AppConfig{
		Name: name,
		Port: 8080,
		Tags: tags,
	}
	if err := storage.SaveConfig(appDir, cfg); err != nil {
		t.Fatal(err)
	}
	return cfg
}

func TestTagAddAndDuplicate(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	setupTagTestApp(t, tmpDir, "app1", nil)

	cmd := NewTagCmd()
	cmd.SetArgs([]string{"add", "app1", "client-a", "prod"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	appDir := filepath.Join(tmpDir, ".local", "share", "gare", "apps", "app1")
	cfg, err := storage.LoadConfig(appDir)
	if err != nil || len(cfg.Tags) != 2 || cfg.Tags[0] != "client-a" || cfg.Tags[1] != "prod" {
		t.Fatalf("expected tags [client-a prod], got: %+v", cfg.Tags)
	}

	dupCmd := NewTagCmd()
	dupCmd.SetArgs([]string{"add", "app1", "PROD", "api"})
	if err := dupCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cfg, _ = storage.LoadConfig(appDir)
	if len(cfg.Tags) != 3 || cfg.Tags[2] != "api" {
		t.Fatalf("expected 3 tags, got: %+v", cfg.Tags)
	}
}

func TestTagAddInvalid(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	setupTagTestApp(t, tmpDir, "app1", nil)

	nonExistCmd := NewTagCmd()
	nonExistCmd.SetArgs([]string{"add", "ghost", "tag1"})
	if err := nonExistCmd.Execute(); err == nil {
		t.Fatal("expected error for non-existent app")
	}

	invalidCmd := NewTagCmd()
	invalidCmd.SetArgs([]string{"add", "app1", "bad tag with spaces"})
	if err := invalidCmd.Execute(); err == nil {
		t.Fatal("expected error for invalid tag")
	}
}

func TestTagRemove(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	setupTagTestApp(t, tmpDir, "app1", []string{"client-a", "prod", "api"})

	rmCmd := NewTagCmd()
	rmCmd.SetArgs([]string{"rm", "app1", "prod", "api"})
	if err := rmCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	appDir := filepath.Join(tmpDir, ".local", "share", "gare", "apps", "app1")
	cfg, _ := storage.LoadConfig(appDir)
	if len(cfg.Tags) != 1 || cfg.Tags[0] != "client-a" {
		t.Fatalf("expected only [client-a], got: %+v", cfg.Tags)
	}
}

func TestTagListAll(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	setupTagTestApp(t, tmpDir, "app1", []string{"client-a", "prod"})
	setupTagTestApp(t, tmpDir, "app2", []string{"client-a", "staging"})

	var buf bytes.Buffer
	cmd := NewTagCmd()
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"list", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var items []tagSummaryItem
	if err := json.Unmarshal(buf.Bytes(), &items); err != nil {
		t.Fatalf("invalid json: %v\noutput: %s", err, buf.String())
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 tag summary items, got: %+v", items)
	}

	buf.Reset()
	cmd.SetArgs([]string{"ls"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "client-a") || !strings.Contains(buf.String(), "prod") {
		t.Fatalf("expected table to contain tags, got: %s", buf.String())
	}
}

func TestTagListAppFiltered(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	setupTagTestApp(t, tmpDir, "app1", []string{"client-a", "prod"})

	var buf bytes.Buffer
	cmd := NewTagCmd()
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"list", "app1", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var items []appTagItem
	if err := json.Unmarshal(buf.Bytes(), &items); err != nil {
		t.Fatalf("invalid json: %v\noutput: %s", err, buf.String())
	}
	if len(items) != 2 || items[0].Tag != "client-a" {
		t.Fatalf("expected 2 tags for app1, got: %+v", items)
	}
}

func TestTagViaAppCmdAndListFilter(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	setupTagTestApp(t, tmpDir, "app1", []string{"client-a", "prod"})
	setupTagTestApp(t, tmpDir, "app2", []string{"client-b", "prod"})

	root := newRootCmd("0.1.0")
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{"list", "--tag", "client-a", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var items []appListItem
	if err := json.Unmarshal(buf.Bytes(), &items); err != nil {
		t.Fatalf("invalid json: %v\noutput: %s", err, buf.String())
	}
	if len(items) != 1 || items[0].Name != "app1" {
		t.Fatalf("expected only app1 for tag client-a, got: %+v", items)
	}
}
