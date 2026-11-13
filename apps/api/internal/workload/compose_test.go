package workload

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FacileStudio/gare/apps/api/internal/storage"
	"github.com/FacileStudio/gare/apps/api/internal/systemd"
	"github.com/FacileStudio/gare/apps/api/internal/testkit"
)

func TestComposeCreateWritesItsUnit(t *testing.T) {
	appDir, repoDir := setupComposeApp(t, "8590:80")
	cfg := &storage.AppConfig{Name: "myapp", RepoURL: "https://example.com/repo.git", AppType: "compose", Port: 8590}
	if err := (composeStack{}).Prepare(context.Background(), testApp(t, cfg, appDir), Create); err != nil {
		t.Fatalf("preparing a compose app failed: %v", err)
	}

	assertComposeUnit(t, repoDir)
	if cfg.ComposeFile != "docker-compose.yml" {
		t.Errorf("expected the discovered compose file to be recorded, got %+v", cfg)
	}
	testkit.AssertGone(t, storage.GetManifestPath(appDir))
}

func TestComposeCreateRetiresLegacyQuadletSources(t *testing.T) {
	appDir, _ := setupComposeApp(t, "8590:80")
	seedLegacyQuadletSources(t, "myapp")
	cfg := &storage.AppConfig{Name: "myapp", RepoURL: "https://example.com/repo.git", AppType: "compose", Port: 8590}
	if err := (composeStack{}).Prepare(context.Background(), testApp(t, cfg, appDir), Create); err != nil {
		t.Fatalf("preparing a compose app failed: %v", err)
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

func TestComposeCreateRejectsUnpublishedPort(t *testing.T) {
	appDir, _ := setupComposeApp(t, "8590:80")
	cfg := &storage.AppConfig{Name: "myapp", RepoURL: "https://example.com/repo.git", AppType: "compose", Port: 8600}
	if err := (composeStack{}).Prepare(context.Background(), testApp(t, cfg, appDir), Create); err == nil {
		t.Error("expected a port mismatch error")
	}
}

func setupComposeApp(t *testing.T, portMapping string) (string, string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	appDir := filepath.Join(home, "apps", "myapp")
	repoDir := storage.GetRepoDir(appDir)
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeRepoFile(t, repoDir, "docker-compose.yml", "services:\n  web:\n    ports:\n      - \""+portMapping+"\"\n")
	return appDir, repoDir
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
