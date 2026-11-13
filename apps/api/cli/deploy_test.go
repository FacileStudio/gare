package cli

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/FacileStudio/gare/apps/api/internal/storage"
)

func TestDeployWaitsForTheAppLock(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	if err := storage.Default().Save(&storage.AppConfig{Name: "app", Branch: "main"}); err != nil {
		t.Fatal(err)
	}
	unlock, err := storage.Default().Lock(t.Context(), "app")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- RunDeploy(t.Context(), "app") }()
	select {
	case err := <-done:
		t.Fatalf("a deploy ran while another held the app lock: %v", err)
	case <-time.After(400 * time.Millisecond):
	}
	unlock()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("the deploy did not proceed once the lock was released")
	}
}
