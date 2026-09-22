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

func TestWriteContainerUnit(t *testing.T) {
	skipWithoutQuadlet(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	appDir := t.TempDir()
	manifestPath := storage.GetManifestPath(appDir)
	if err := os.WriteFile(manifestPath, []byte("apiVersion: v1\nkind: Pod\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := writeContainerUnit(context.Background(), "myapp", appDir); err != nil {
		t.Fatalf("writeContainerUnit failed: %v", err)
	}
	data, err := os.ReadFile(quadlet.KubePath("myapp"))
	if err != nil {
		t.Fatalf("quadlet source was not written: %v", err)
	}
	if !strings.Contains(string(data), "Yaml=\""+manifestPath+"\"") {
		t.Errorf("quadlet source is missing the manifest path:\n%s", string(data))
	}
	if !strings.Contains(string(data), "ExitCodePropagation=any") {
		t.Errorf("quadlet source must propagate container failure to systemd:\n%s", string(data))
	}
}

func TestWriteContainerUnitRejectsForeignUnitFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	appDir := t.TempDir()
	unitPath := writeUnitFile(t, "myapp", "[Service]\nDescription=Hand written by the operator\nExecStart=/usr/local/bin/myapp\n")

	err := writeContainerUnit(context.Background(), "myapp", appDir)
	if err == nil {
		t.Fatal("expected an error when a unit file gare does not own would shadow the quadlet unit")
	}
	if !strings.Contains(err.Error(), unitPath) {
		t.Errorf("error should name the shadowing file, got %v", err)
	}
	if _, statErr := os.Stat(quadlet.KubePath("myapp")); !os.IsNotExist(statErr) {
		t.Error("a shadowed workload must not write a quadlet source")
	}
	assertExists(t, unitPath)
}

func TestWriteContainerUnitRetiresGareUnitFile(t *testing.T) {
	skipWithoutQuadlet(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	appDir := t.TempDir()
	unitPath := writeUnitFile(t, "myapp", "[Unit]\nDescription=Gare Managed App: myapp\n\n[Service]\nExecStart=/usr/bin/podman kube play old.yaml\n")
	writeEnableLink(t, "myapp", unitPath)

	if err := writeContainerUnit(context.Background(), "myapp", appDir); err != nil {
		t.Fatalf("an upgrade must not cost the application, got: %v", err)
	}
	if _, err := os.Stat(quadlet.KubePath("myapp")); err != nil {
		t.Errorf("expected the quadlet source to be written: %v", err)
	}
	assertGone(t, unitPath)
	assertGone(t, enableLinkPath("myapp"))
}

func TestWriteContainerUnitRetiresOwnComposeUnitFile(t *testing.T) {
	skipWithoutQuadlet(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	appDir := t.TempDir()
	unitPath := writeUnitFile(t, "myapp", "[Unit]\nDescription=Gare Managed Compose App: myapp\n\n[Service]\nExecStart=/usr/bin/podman compose up -d\n")

	if err := writeContainerUnit(context.Background(), "myapp", appDir); err != nil {
		t.Fatalf("a workload type change must not cost the application, got: %v", err)
	}
	assertGone(t, unitPath)
}

func TestWriteStaticUnit(t *testing.T) {
	skipWithoutQuadlet(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	appDir := t.TempDir()
	rootDir := filepath.Join(storage.GetRepoDir(appDir), "dist")

	if err := writeStaticUnit(context.Background(), "blog", appDir, rootDir, 8100); err != nil {
		t.Fatalf("writeStaticUnit failed: %v", err)
	}
	data, err := os.ReadFile(quadlet.ContainerPath("blog"))
	if err != nil {
		t.Fatalf("quadlet source was not written: %v", err)
	}
	for _, snippet := range []string{
		"Image=docker.io/library/caddy:2-alpine",
		"PublishPort=8100:80",
		"Volume=" + rootDir + ":/srv:ro,Z",
		"Volume=" + storage.GetAppStaticConfigPath(appDir) + ":/etc/caddy/Caddyfile:ro,Z",
		"EnvironmentFile=" + storage.GetAppEnvPath(appDir),
	} {
		if !strings.Contains(string(data), snippet) {
			t.Errorf("static source is missing %q:\n%s", snippet, string(data))
		}
	}
	if _, err := os.Stat(storage.GetAppEnvPath(appDir)); err != nil {
		t.Errorf("the env file must exist for the container source to load it: %v", err)
	}
	exists, err := appUnitExists("blog")
	if err != nil || !exists {
		t.Errorf("expected the static source to count as deployed artifacts, got exists=%v err=%v", exists, err)
	}
}

func TestWriteStaticUnitKeepsSPAFallback(t *testing.T) {
	skipWithoutQuadlet(t)
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

func TestWriteStaticUnitRejectsForeignUnitFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	appDir := t.TempDir()
	unitPath := writeUnitFile(t, "blog", "[Service]\nExecStart=/usr/bin/caddy file-server\n")

	err := writeStaticUnit(context.Background(), "blog", appDir, filepath.Join(appDir, "dist"), 8100)
	if err == nil {
		t.Fatal("expected an error when a unit file gare does not own would shadow the quadlet unit")
	}
	if !strings.Contains(err.Error(), unitPath) {
		t.Errorf("error should name the shadowing file, got %v", err)
	}
	if _, statErr := os.Stat(quadlet.ContainerPath("blog")); !os.IsNotExist(statErr) {
		t.Error("a shadowed workload must not write a quadlet source")
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

	if err := os.MkdirAll(quadlet.Dir(), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(quadlet.KubePath("myapp"), []byte("[Kube]\n"), 0644); err != nil {
		t.Fatal(err)
	}
	exists, err = appUnitExists("myapp")
	if err != nil || !exists {
		t.Errorf("expected a quadlet source to count as deployed artifacts, got exists=%v err=%v", exists, err)
	}
}

func skipWithoutQuadlet(t *testing.T) {
	t.Helper()
	if err := quadlet.CheckGenerator(); err != nil {
		t.Skipf("quadlet generator is not installed: %v", err)
	}
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

func writeEnableLink(t *testing.T, name, unitPath string) {
	t.Helper()
	linkPath := enableLinkPath(name)
	if err := os.MkdirAll(filepath.Dir(linkPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(unitPath, linkPath); err != nil {
		t.Fatal(err)
	}
}

func enableLinkPath(name string) string {
	return filepath.Join(systemd.DefaultUserUnitDir(), "default.target.wants", name+".service")
}

func assertGone(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Errorf("expected %s to be removed, got %v", path, err)
	}
}

func assertExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected %s to be left in place, got %v", path, err)
	}
}
