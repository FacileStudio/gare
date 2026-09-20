package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/FacileStudio/gare/internal/storage"
)

func TestLifecycleUnitNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	appDir := filepath.Join(tmpDir, ".local", "share", "gare", "apps", "testcontainer")
	cfg := &storage.AppConfig{Name: "testcontainer", AppType: "container"}
	if err := storage.SaveConfig(appDir, cfg); err != nil {
		t.Fatal(err)
	}

	startCmd := NewStartCmd()
	startCmd.SetArgs([]string{"testcontainer"})
	if err := startCmd.Execute(); err == nil {
		t.Fatalf("expected error when unit is not found in systemd, got nil")
	}

	stopCmd := NewStopCmd()
	stopCmd.SetArgs([]string{"testcontainer"})
	if err := stopCmd.Execute(); err == nil {
		t.Fatalf("expected error when unit is not found in systemd, got nil")
	}
}

func TestLifecycleAppNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	startCmd := NewStartCmd()
	startCmd.SetArgs([]string{"nonexistent"})
	err := startCmd.Execute()
	if err == nil {
		t.Fatal("expected error for nonexistent app, got nil")
	}
	if !strings.Contains(err.Error(), "gare list") {
		t.Errorf("error must tell the user how to list apps, got %v", err)
	}
}
