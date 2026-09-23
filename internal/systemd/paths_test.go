package systemd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLegacyQuadletSourcesUseXDGConfigHome(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	sources := LegacyQuadletSources("myapp")
	dir := filepath.Join(configHome, "containers", "systemd")
	want := []string{
		filepath.Join(dir, "myapp.kube"),
		filepath.Join(dir, "myapp.container"),
	}
	if len(sources) != len(want) {
		t.Fatalf("LegacyQuadletSources() returned %d paths, want %d", len(sources), len(want))
	}
	for i, path := range want {
		if sources[i] != path {
			t.Errorf("LegacyQuadletSources()[%d] = %q, want %q", i, sources[i], path)
		}
	}
}

func TestLegacyQuadletSourcesExist(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	exists, err := LegacyQuadletSourcesExist("myapp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Error("expected no legacy source for an app that never had one")
	}

	dir := filepath.Dir(LegacyQuadletSources("myapp")[0])
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(LegacyQuadletSources("myapp")[1], []byte("[Container]\n"), 0644); err != nil {
		t.Fatal(err)
	}
	exists, err = LegacyQuadletSourcesExist("myapp")
	if err != nil || !exists {
		t.Errorf("expected the container source to be detected, got exists=%v err=%v", exists, err)
	}
}

func TestRemoveLegacyQuadletSources(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	if err := RemoveLegacyQuadletSources("myapp"); err != nil {
		t.Fatalf("removing missing sources returned an error: %v", err)
	}

	for _, path := range LegacyQuadletSources("myapp") {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("[Kube]\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := RemoveLegacyQuadletSources("myapp"); err != nil {
		t.Fatalf("RemoveLegacyQuadletSources failed: %v", err)
	}
	for _, path := range LegacyQuadletSources("myapp") {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("expected %s to be removed, got %v", path, err)
		}
	}
}
