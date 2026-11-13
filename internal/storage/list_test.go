package storage

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func writeBrokenApp(t *testing.T, baseDir, name string) {
	t.Helper()
	dir := filepath.Join(baseDir, name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(GetConfigPath(dir), []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestListAppsStrictNamesBrokenApps(t *testing.T) {
	baseDir := t.TempDir()
	if err := SaveConfig(filepath.Join(baseDir, "ok"), &AppConfig{Name: "ok", Port: 8000}); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(baseDir, "creating"), 0755); err != nil {
		t.Fatal(err)
	}
	if apps, err := ListAppsStrict(baseDir); err != nil || len(apps) != 1 {
		t.Fatalf("a directory without config is not an app: %d apps, err %v", len(apps), err)
	}
	writeBrokenApp(t, baseDir, "broken")
	if _, err := ListAppsStrict(baseDir); !errors.Is(err, ErrUnreadableApps) {
		t.Fatalf("expected ErrUnreadableApps, got %v", err)
	}
	if apps, err := ListApps(baseDir); err != nil || len(apps) != 1 {
		t.Fatalf("ListApps must still list readable apps: %d, err %v", len(apps), err)
	}
	if broken, _ := UnreadableApps(baseDir); len(broken) != 1 || broken[0] != "broken" {
		t.Fatalf("UnreadableApps: %v", broken)
	}
}

func TestDiscoverAvailablePortRefusesWithUnreadableApp(t *testing.T) {
	baseDir := t.TempDir()
	writeBrokenApp(t, baseDir, "broken")
	if _, err := DiscoverAvailablePort(baseDir, 0); !errors.Is(err, ErrUnreadableApps) {
		t.Fatalf("expected port discovery to refuse, got %v", err)
	}
}
