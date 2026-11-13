package workload

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FacileStudio/gare/apps/api/internal/storage"
	"github.com/FacileStudio/gare/apps/api/internal/systemd"
)

func TestWriteContainerUnit(t *testing.T) {
	skipWithoutPodmanWorkloads(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	appDir := t.TempDir()
	manifestPath := storage.GetManifestPath(appDir)
	if err := os.WriteFile(manifestPath, []byte("apiVersion: v1\nkind: Pod\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := writeContainerUnit(context.Background(), testApp(t, &storage.AppConfig{Name: "myapp"}, appDir)); err != nil {
		t.Fatalf("writeContainerUnit failed: %v", err)
	}
	data, err := os.ReadFile(systemd.GetUnitPath("myapp"))
	if err != nil {
		t.Fatalf("unit was not written: %v", err)
	}
	if !strings.Contains(string(data), "kube play") || !strings.Contains(string(data), manifestPath) {
		t.Errorf("unit is missing the manifest path or the play command:\n%s", string(data))
	}
	if !strings.Contains(string(data), "--service-exit-code-propagation=any") {
		t.Errorf("unit must propagate container failure to systemd:\n%s", string(data))
	}
}

func TestWriteStaticUnit(t *testing.T) {
	skipWithoutPodmanWorkloads(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	appDir := t.TempDir()
	rootDir := filepath.Join(storage.GetRepoDir(appDir), "dist")

	if err := writeStaticUnit(context.Background(), testApp(t, &storage.AppConfig{Name: "blog", Port: 8100}, appDir), rootDir); err != nil {
		t.Fatalf("writeStaticUnit failed: %v", err)
	}
	data, err := os.ReadFile(systemd.GetUnitPath("blog"))
	if err != nil {
		t.Fatalf("unit was not written: %v", err)
	}
	for _, snippet := range []string{
		"docker.io/library/caddy:2-alpine",
		"-v " + rootDir + ":/srv:ro,Z",
		"-v " + storage.GetAppStaticConfigPath(appDir) + ":/etc/caddy/Caddyfile:ro,Z",
		"--publish 8100:80",
		"--env-file " + storage.GetAppEnvPath(appDir),
	} {
		if !strings.Contains(string(data), snippet) {
			t.Errorf("static unit is missing %q:\n%s", snippet, string(data))
		}
	}
	if _, err := os.Stat(storage.GetAppEnvPath(appDir)); err != nil {
		t.Errorf("the env file must exist for the unit to load it: %v", err)
	}
}

func TestWriteStaticUnitKeepsSPAFallback(t *testing.T) {
	skipWithoutPodmanWorkloads(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	appDir := t.TempDir()

	if err := writeStaticUnit(context.Background(), testApp(t, &storage.AppConfig{Name: "blog", Port: 8100}, appDir), filepath.Join(appDir, "dist")); err != nil {
		t.Fatalf("writeStaticUnit failed: %v", err)
	}
	config, err := os.ReadFile(storage.GetAppStaticConfigPath(appDir))
	if err != nil {
		t.Fatalf("the Caddyfile the container serves was not written: %v", err)
	}
	if !strings.Contains(string(config), "try_files {path} /index.html") {
		t.Errorf("static deep links regress without the SPA fallback:\n%s", string(config))
	}
}
