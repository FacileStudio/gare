package quadlet

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDirUsesXDGConfigHome(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	expected := filepath.Join(configHome, "containers", "systemd")
	if got := Dir(); got != expected {
		t.Errorf("Dir() = %q, want %q", got, expected)
	}
}

func TestDirFallsBackToHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", home)

	expected := filepath.Join(home, ".config", "containers", "systemd")
	if got := Dir(); got != expected {
		t.Errorf("Dir() = %q, want %q", got, expected)
	}
}

func TestDirMatchesTheSystemdUserConfigHome(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	if !strings.HasPrefix(Dir(), configHome) {
		t.Errorf("Dir() = %q, want it under %q", Dir(), configHome)
	}
}

func TestKubePath(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	expected := filepath.Join(configHome, "containers", "systemd", "myapp.kube")
	if got := KubePath("myapp"); got != expected {
		t.Errorf("KubePath() = %q, want %q", got, expected)
	}
}

func TestRemove(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := Remove("missing"); err != nil {
		t.Fatalf("Remove on a missing source returned an error: %v", err)
	}

	if err := os.MkdirAll(Dir(), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(KubePath("myapp"), []byte("[Kube]\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := Remove("myapp"); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}
	if _, err := os.Stat(KubePath("myapp")); !os.IsNotExist(err) {
		t.Error("expected the quadlet source to be removed")
	}
}

func runQuadletDryRun(t *testing.T, unitDir string) string {
	t.Helper()
	generator, err := GeneratorPath()
	if err != nil {
		t.Skipf("quadlet generator is not installed: %v", err)
	}
	cmd := exec.Command(generator, "--user", "--dryrun")
	cmd.Env = append(os.Environ(), "QUADLET_UNIT_DIRS="+unitDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("quadlet rejected the generated source: %v\n%s", err, output)
	}
	return string(output)
}

func TestGeneratorPathFromPrefersFirstInstalledCandidate(t *testing.T) {
	dir := t.TempDir()
	installed := filepath.Join(dir, "podman-user-generator")
	if err := os.WriteFile(installed, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(dir, "podman-system-generator")

	got, err := generatorPathFrom([]string{missing, installed})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != installed {
		t.Errorf("generatorPathFrom() = %q, want %q", got, installed)
	}
}

func TestGeneratorPathFromSkipsDirectories(t *testing.T) {
	dir := t.TempDir()
	asDirectory := filepath.Join(dir, "quadlet")
	if err := os.MkdirAll(asDirectory, 0755); err != nil {
		t.Fatal(err)
	}

	if _, err := generatorPathFrom([]string{asDirectory}); err == nil {
		t.Error("expected a directory to not count as an installed generator")
	}
}

func TestGeneratorPathFromReportsSearchedPaths(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "podman-user-generator")
	_, err := generatorPathFrom([]string{missing})
	if err == nil {
		t.Fatal("expected an error when no candidate exists")
	}
	if !strings.Contains(err.Error(), missing) {
		t.Errorf("error should name the searched path, got %v", err)
	}
}
