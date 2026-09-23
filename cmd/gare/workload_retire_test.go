package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FacileStudio/gare/internal/systemd"
)

func TestWriteContainerUnitRejectsForeignUnitFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	appDir := t.TempDir()
	const foreign = "[Service]\nDescription=Hand written by the operator\nExecStart=/usr/local/bin/myapp\n"
	unitPath := writeUnitFile(t, "myapp", foreign)

	err := writeContainerUnit(context.Background(), "myapp", appDir)
	if err == nil {
		t.Fatal("expected an error when a unit file gare does not own already uses the name")
	}
	if !strings.Contains(err.Error(), unitPath) {
		t.Errorf("error should name the unit file gare refused to overwrite, got %v", err)
	}
	assertUnitContent(t, unitPath, foreign)
}

func TestWriteStaticUnitRejectsForeignUnitFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	appDir := t.TempDir()
	const foreign = "[Service]\nExecStart=/usr/bin/caddy file-server\n"
	unitPath := writeUnitFile(t, "blog", foreign)

	err := writeStaticUnit(context.Background(), "blog", appDir, filepath.Join(appDir, "dist"), 8100)
	if err == nil {
		t.Fatal("expected an error when a unit file gare does not own already uses the name")
	}
	if !strings.Contains(err.Error(), unitPath) {
		t.Errorf("error should name the unit file gare refused to overwrite, got %v", err)
	}
	assertUnitContent(t, unitPath, foreign)
}

func TestWriteContainerUnitReplacesGareOwnedUnitFile(t *testing.T) {
	skipWithoutPodmanWorkloads(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	appDir := t.TempDir()
	compose := "[Unit]\nDescription=" + systemd.ComposeUnitDescription("myapp") + "\n\n[Service]\nExecStart=/usr/bin/podman compose up -d\n"
	writeUnitFile(t, "myapp", compose)

	previous, err := guardUnitFile("myapp")
	if err != nil {
		t.Fatalf("a unit gare wrote must be replaceable: %v", err)
	}
	if !replacedWorkloadType(previous, systemd.KubeUnitDescription("myapp")) {
		t.Fatal("a workload type change must retire the workload it replaces, or the outgoing stack stays up")
	}
	if err := writeContainerUnit(context.Background(), "myapp", appDir); err != nil {
		t.Fatalf("a workload type change must not cost the application, got: %v", err)
	}
	data, err := os.ReadFile(systemd.GetUnitPath("myapp"))
	if err != nil {
		t.Fatalf("expected the unit to be rewritten: %v", err)
	}
	if !strings.Contains(string(data), "Description="+systemd.KubeUnitDescription("myapp")+"\n") {
		t.Errorf("expected the compose unit to be replaced in place:\n%s", string(data))
	}
}

func TestWriteContainerUnitRetiresLegacyQuadletSources(t *testing.T) {
	skipWithoutPodmanWorkloads(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	appDir := t.TempDir()
	for _, path := range systemd.LegacyQuadletSources("myapp") {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("[Kube]\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	if err := writeContainerUnit(context.Background(), "myapp", appDir); err != nil {
		t.Fatalf("writeContainerUnit failed: %v", err)
	}
	for _, path := range systemd.LegacyQuadletSources("myapp") {
		assertGone(t, path)
	}
}

func TestGuardUnitFileIdentifiesTheOccupyingWorkload(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cases := map[string]string{
		"container": "[Unit]\nDescription=" + systemd.KubeUnitDescription("myapp") + "\n",
		"static":    "[Unit]\nDescription=" + systemd.StaticUnitDescription("myapp") + "\n",
		"compose":   "[Unit]\nDescription=" + systemd.ComposeUnitDescription("myapp") + "\n",
	}
	for name, content := range cases {
		writeUnitFile(t, "myapp", content)
		got, err := guardUnitFile("myapp")
		if err != nil {
			t.Fatalf("expected the %s unit to be recognised, got %v", name, err)
		}
		if !strings.Contains(content, "Description="+got) {
			t.Errorf("guardUnitFile reported %q, which is not the %s unit's description", got, name)
		}
	}
}

func TestGuardUnitFileRejectsAnotherAppsUnit(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	writeUnitFile(t, "myapp", "[Unit]\nDescription="+systemd.KubeUnitDescription("otherapp")+"\n")

	if _, err := guardUnitFile("myapp"); err == nil {
		t.Error("expected a unit describing another application to be refused")
	}
}

func TestGuardUnitFileReportsAFreeName(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	previous, err := guardUnitFile("undeployed")
	if err != nil || previous != "" {
		t.Errorf("expected a free unit name to report no previous workload, got %q err=%v", previous, err)
	}
}

func TestReplacedWorkloadType(t *testing.T) {
	compose := systemd.ComposeUnitDescription("myapp")
	kube := systemd.KubeUnitDescription("myapp")
	cases := []struct {
		previous string
		current  string
		want     bool
	}{
		{previous: compose, current: kube, want: true},
		{previous: kube, current: kube, want: false},
		{previous: compose, current: compose, want: false},
		{previous: "", current: kube, want: false},
	}
	for _, tc := range cases {
		if got := replacedWorkloadType(tc.previous, tc.current); got != tc.want {
			t.Errorf("replacedWorkloadType(%q, %q) = %v, want %v", tc.previous, tc.current, got, tc.want)
		}
	}
}
