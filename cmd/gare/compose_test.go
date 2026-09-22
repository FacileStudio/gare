package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FacileStudio/gare/internal/quadlet"
	"github.com/FacileStudio/gare/internal/storage"
	"github.com/FacileStudio/gare/internal/systemd"
)

func TestWriteComposeArtifacts(t *testing.T) {
	_, appDir, repoDir := setupComposeApp(t, "8590:80")
	opts := appCreateOptions{repo: "https://example.com/repo.git", appType: "compose", port: 8590}
	if err := writeComposeArtifacts(context.Background(), "myapp", appDir, opts); err != nil {
		t.Fatalf("writeComposeArtifacts failed: %v", err)
	}

	assertComposeUnit(t, repoDir)
	assertComposeConfig(t, appDir)
}

func TestWriteComposeArtifactsRetiresQuadletSources(t *testing.T) {
	_, appDir, _ := setupComposeApp(t, "8590:80")
	if err := quadlet.WriteKubeUnit("myapp", storage.GetManifestPath(appDir)); err != nil {
		t.Fatalf("WriteKubeUnit failed: %v", err)
	}
	opts := appCreateOptions{repo: "https://example.com/repo.git", appType: "compose", port: 8590}
	if err := writeComposeArtifacts(context.Background(), "myapp", appDir, opts); err != nil {
		t.Fatalf("writeComposeArtifacts failed: %v", err)
	}

	for _, path := range []string{quadlet.KubePath("myapp"), quadlet.ContainerPath("myapp")} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("a workload type change must not leave %s behind", path)
		}
	}
	if _, err := os.Stat(systemd.GetUnitPath("myapp")); err != nil {
		t.Errorf("expected the synthesized compose unit to exist: %v", err)
	}
}

func TestWriteComposeArtifactsRejectsUnpublishedPort(t *testing.T) {
	_, appDir, _ := setupComposeApp(t, "8590:80")
	opts := appCreateOptions{repo: "https://example.com/repo.git", appType: "compose", port: 8600}
	if err := writeComposeArtifacts(context.Background(), "myapp", appDir, opts); err == nil {
		t.Error("expected a port mismatch error")
	}
}

func setupComposeApp(t *testing.T, portMapping string) (string, string, string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	appDir := filepath.Join(home, "app")
	repoDir := storage.GetRepoDir(appDir)
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeRepoComposeFile(t, repoDir, "docker-compose.yml", "services:\n  web:\n    ports:\n      - \""+portMapping+"\"\n")
	return home, appDir, repoDir
}

func assertComposeUnit(t *testing.T, repoDir string) {
	t.Helper()
	data, err := os.ReadFile(systemd.GetUnitPath("myapp"))
	if err != nil {
		t.Fatalf("compose unit was not written: %v", err)
	}
	unit := string(data)
	if !strings.Contains(unit, "compose -f docker-compose.yml -p myapp up") {
		t.Errorf("expected compose unit command, got:\n%s", unit)
	}
	if !strings.Contains(unit, "WorkingDirectory="+repoDir) {
		t.Errorf("expected repo working directory, got:\n%s", unit)
	}
}

func assertComposeConfig(t *testing.T, appDir string) {
	t.Helper()
	cfg, err := storage.LoadConfig(appDir)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	if !cfg.IsCompose() || cfg.ComposeFile != "docker-compose.yml" || cfg.Port != 8590 {
		t.Errorf("unexpected stored config: %+v", cfg)
	}
	if _, err := os.Stat(storage.GetManifestPath(appDir)); !os.IsNotExist(err) {
		t.Error("compose workloads must not generate a pod manifest")
	}
}
