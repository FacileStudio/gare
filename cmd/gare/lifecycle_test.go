package main

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FacileStudio/gare/internal/storage"
)

func TestLifecycleUnitNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	appDir := filepath.Join(tmpDir, ".local", "share", "gare", "apps", "testcontainer")
	cfg := &storage.AppConfig{Name: "testcontainer", AppType: "compose"}
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

func TestStartContainerImageMissing(t *testing.T) {
	if _, err := exec.LookPath("podman"); err != nil {
		t.Skip("podman not found in PATH")
	}
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	name := "gare-missing-image-test"
	appDir := filepath.Join(tmpDir, ".local", "share", "gare", "apps", name)
	cfg := &storage.AppConfig{Name: name, AppType: "container", Port: 8000}
	if err := storage.SaveConfig(appDir, cfg); err != nil {
		t.Fatal(err)
	}

	startCmd := NewStartCmd()
	startCmd.SetArgs([]string{name})
	err := startCmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "gare deploy") {
		t.Fatalf("expected an actionable missing-image error, got %v", err)
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
