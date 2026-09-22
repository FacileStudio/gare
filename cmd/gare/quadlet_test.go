package main

import (
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

	if err := writeContainerUnit("myapp", appDir); err != nil {
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

func TestWriteContainerUnitRejectsShadowingUnitFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", "")
	appDir := t.TempDir()
	unitPath := systemd.GetUnitPath("myapp")
	if err := os.MkdirAll(filepath.Dir(unitPath), 0755); err != nil {
		t.Fatal(err)
	}
	shadow := "[Service]\nDescription=Gare Managed Static App: myapp\nExecStart=/usr/bin/caddy file-server\n"
	if err := os.WriteFile(unitPath, []byte(shadow), 0644); err != nil {
		t.Fatal(err)
	}

	err := writeContainerUnit("myapp", appDir)
	if err == nil {
		t.Fatal("expected an error when a unit file would shadow the quadlet unit")
	}
	if !strings.Contains(err.Error(), unitPath) {
		t.Errorf("error should name the shadowing file, got %v", err)
	}
	if _, statErr := os.Stat(quadlet.KubePath("myapp")); !os.IsNotExist(statErr) {
		t.Error("a shadowed workload must not write a quadlet source")
	}
}

func TestWriteStaticUnit(t *testing.T) {
	skipWithoutQuadlet(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	appDir := t.TempDir()
	rootDir := filepath.Join(storage.GetRepoDir(appDir), "dist")

	if err := writeStaticUnit("blog", appDir, rootDir, 8100); err != nil {
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

	if err := writeStaticUnit("blog", appDir, filepath.Join(appDir, "dist"), 8100); err != nil {
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

func TestWriteStaticUnitRejectsShadowingUnitFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", "")
	appDir := t.TempDir()
	unitPath := systemd.GetUnitPath("blog")
	if err := os.MkdirAll(filepath.Dir(unitPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(unitPath, []byte("[Service]\nExecStart=/usr/bin/caddy file-server\n"), 0644); err != nil {
		t.Fatal(err)
	}

	err := writeStaticUnit("blog", appDir, filepath.Join(appDir, "dist"), 8100)
	if err == nil {
		t.Fatal("expected an error when a unit file would shadow the quadlet unit")
	}
	if _, statErr := os.Stat(quadlet.ContainerPath("blog")); !os.IsNotExist(statErr) {
		t.Error("a shadowed workload must not write a quadlet source")
	}
}

func TestAppUnitExists(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", "")

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
