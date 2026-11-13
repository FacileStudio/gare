package systemd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultUserUnitDirFollowsXDGConfigHome(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	expected := filepath.Join(configHome, "systemd", "user")
	if got := DefaultUserUnitDir(); got != expected {
		t.Errorf("DefaultUserUnitDir() = %q, want %q", got, expected)
	}
	if got := GetUnitPath("myapp"); got != filepath.Join(expected, "myapp.service") {
		t.Errorf("GetUnitPath() = %q, want %q", got, filepath.Join(expected, "myapp.service"))
	}
}

func TestDefaultUserUnitDirFallsBackToHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", home)

	expected := filepath.Join(home, ".config", "systemd", "user")
	if got := DefaultUserUnitDir(); got != expected {
		t.Errorf("DefaultUserUnitDir() = %q, want %q", got, expected)
	}
}

func TestRemoveUnitClearsEnableLink(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	unitPath := GetUnitPath("myapp")
	linkPath := filepath.Join(DefaultUserUnitDir(), "default.target.wants", "myapp.service")
	if err := os.MkdirAll(filepath.Dir(linkPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(unitPath, []byte("[Unit]\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(unitPath, linkPath); err != nil {
		t.Fatal(err)
	}

	if err := RemoveUnit("myapp"); err != nil {
		t.Fatalf("RemoveUnit failed: %v", err)
	}
	for _, path := range []string{unitPath, linkPath} {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Errorf("expected %s to be removed, got %v", path, err)
		}
	}
}

func TestRemoveUnitToleratesMissingArtifacts(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	if err := RemoveUnit("never-written"); err != nil {
		t.Errorf("RemoveUnit on a missing unit returned an error: %v", err)
	}
}

// TestGareUnitDescriptionsMatchTheTemplates pins the coupling the deploy guard depends on: a
// description that drifts from its template makes gare refuse to redeploy its own unit.
func TestGareUnitDescriptionsMatchTheTemplates(t *testing.T) {
	const name = "my-app"
	kube, err := GenerateKubeUnit(KubeUnitData{Name: name, YamlPath: "/srv/manifest.yaml", PodmanPath: "/usr/bin/podman"})
	if err != nil {
		t.Fatal(err)
	}
	static, err := GenerateStaticUnit(withPodmanPath(StaticSiteUnit(name, 8100, "/srv/dist", "/srv/env", "/srv/Caddyfile")))
	if err != nil {
		t.Fatal(err)
	}
	compose, err := GenerateComposeUnit(composeUnitInput(name))
	if err != nil {
		t.Fatal(err)
	}

	units := []struct {
		unit        string
		description string
	}{
		{kube, KubeUnitDescription(name)},
		{static, StaticUnitDescription(name)},
		{compose, ComposeUnitDescription(name)},
	}
	for _, tc := range units {
		if !strings.Contains(tc.unit, "Description="+tc.description+"\n") {
			t.Errorf("expected the unit to declare %q, got:\n%s", tc.description, tc.unit)
		}
	}
	if got := len(GareUnitDescriptions(name)); got != len(units) {
		t.Errorf("GareUnitDescriptions returned %d descriptions, want %d", got, len(units))
	}
}

func TestResolvePodmanPath(t *testing.T) {
	path := ResolvePodmanPath()
	if path == "" {
		t.Fatal("expected non-empty podman path")
	}
}

func TestUserEnviron(t *testing.T) {
	env := userEnviron()
	hasRuntimeDir := false
	hasDBusBus := false
	for _, e := range env {
		if strings.HasPrefix(e, "XDG_RUNTIME_DIR=") {
			hasRuntimeDir = true
		}
		if strings.HasPrefix(e, "DBUS_SESSION_BUS_ADDRESS=") {
			hasDBusBus = true
		}
	}
	if !hasRuntimeDir {
		t.Error("expected XDG_RUNTIME_DIR to be present in userEnviron")
	}
	if !hasDBusBus {
		t.Error("expected DBUS_SESSION_BUS_ADDRESS to be present in userEnviron")
	}
}
