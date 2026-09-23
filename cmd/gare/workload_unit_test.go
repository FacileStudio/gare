package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FacileStudio/gare/internal/storage"
	"github.com/FacileStudio/gare/internal/systemd"
)

func TestWriteContainerUnit(t *testing.T) {
	skipWithoutPodmanWorkloads(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	appDir := t.TempDir()
	manifestPath := storage.GetManifestPath(appDir)
	if err := os.WriteFile(manifestPath, []byte("apiVersion: v1\nkind: Pod\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := writeContainerUnit(context.Background(), "myapp", appDir); err != nil {
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

	if err := writeStaticUnit(context.Background(), "blog", appDir, rootDir, 8100); err != nil {
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
	exists, err := appUnitExists("blog")
	if err != nil || !exists {
		t.Errorf("expected the static unit to count as deployed artifacts, got exists=%v err=%v", exists, err)
	}
}

func TestWriteStaticUnitKeepsSPAFallback(t *testing.T) {
	skipWithoutPodmanWorkloads(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	appDir := t.TempDir()

	if err := writeStaticUnit(context.Background(), "blog", appDir, filepath.Join(appDir, "dist"), 8100); err != nil {
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

func TestAppUnitExists(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	exists, err := appUnitExists("never-deployed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Error("expected no artifacts for an undeployed app")
	}

	writeUnitFile(t, "myapp", "[Unit]\nDescription="+systemd.KubeUnitDescription("myapp")+"\n")
	exists, err = appUnitExists("myapp")
	if err != nil || !exists {
		t.Errorf("expected a gare-written unit to count as deployed artifacts, got exists=%v err=%v", exists, err)
	}

	legacy := systemd.LegacyQuadletSources("older")[0]
	if err := os.MkdirAll(filepath.Dir(legacy), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacy, []byte("[Kube]\n"), 0644); err != nil {
		t.Fatal(err)
	}
	exists, err = appUnitExists("older")
	if err != nil || !exists {
		t.Errorf("an app deployed by an older gare must still be recognised, got exists=%v err=%v", exists, err)
	}
}

// skipWithoutPodmanWorkloads lets a machine without a usable podman skip these tests, while CI
// fails loudly: the point of them is to prove the generated units actually run, so a silent skip
// there would hide exactly the regression they exist to catch. CI sets GARE_REQUIRE_PODMAN.
func skipWithoutPodmanWorkloads(t *testing.T) {
	t.Helper()
	err := systemd.CheckPodmanWorkloads(context.Background())
	if err == nil {
		return
	}
	if os.Getenv("GARE_REQUIRE_PODMAN") == "1" {
		t.Fatalf("podman cannot run the generated units: %v", err)
	}
	t.Skipf("podman cannot run the generated units here: %v", err)
}

func writeUnitFile(t *testing.T, name, content string) string {
	t.Helper()
	unitPath := systemd.GetUnitPath(name)
	if err := os.MkdirAll(filepath.Dir(unitPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(unitPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return unitPath
}

func assertUnitContent(t *testing.T, unitPath, want string) {
	t.Helper()
	data, err := os.ReadFile(unitPath)
	if err != nil {
		t.Fatalf("expected %s to be left in place, got %v", unitPath, err)
	}
	if string(data) != want {
		t.Errorf("expected %s to keep its own definition, got:\n%s", unitPath, string(data))
	}
}

func assertGone(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Errorf("expected %s to be removed, got %v", path, err)
	}
}
