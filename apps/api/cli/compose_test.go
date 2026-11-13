package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FacileStudio/gare/apps/api/internal/storage"
	"github.com/FacileStudio/gare/apps/api/internal/systemd"
)

func TestWriteComposeArtifacts(t *testing.T) {
	_, appDir, repoDir := setupComposeApp(t, "8590:80")
	cfg := &storage.AppConfig{Name: "myapp", RepoURL: "https://example.com/repo.git", AppType: "compose", Port: 8590}
	if err := writeComposeArtifacts(context.Background(), storage.Default(), cfg); err != nil {
		t.Fatalf("writeComposeArtifacts failed: %v", err)
	}

	assertComposeUnit(t, repoDir)
	assertComposeConfig(t, appDir)
}

func TestWriteComposeArtifactsRetiresLegacyQuadletSources(t *testing.T) {
	setupComposeApp(t, "8590:80")
	for _, path := range systemd.LegacyQuadletSources("myapp") {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("[Kube]\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	cfg := &storage.AppConfig{Name: "myapp", RepoURL: "https://example.com/repo.git", AppType: "compose", Port: 8590}
	if err := writeComposeArtifacts(context.Background(), storage.Default(), cfg); err != nil {
		t.Fatalf("writeComposeArtifacts failed: %v", err)
	}

	for _, path := range systemd.LegacyQuadletSources("myapp") {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("a workload type change must not leave %s behind", path)
		}
	}
	if _, err := os.Stat(systemd.GetUnitPath("myapp")); err != nil {
		t.Errorf("expected the synthesized compose unit to exist: %v", err)
	}
}

func TestWriteComposeArtifactsRejectsUnpublishedPort(t *testing.T) {
	setupComposeApp(t, "8590:80")
	cfg := &storage.AppConfig{Name: "myapp", RepoURL: "https://example.com/repo.git", AppType: "compose", Port: 8600}
	if err := writeComposeArtifacts(context.Background(), storage.Default(), cfg); err == nil {
		t.Error("expected a port mismatch error")
	}
}

func setupComposeApp(t *testing.T, portMapping string) (string, string, string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	appDir := storage.Default().AppDir("myapp")
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
	cfg, err := storage.Open(filepath.Dir(appDir)).Load(filepath.Base(appDir))
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
