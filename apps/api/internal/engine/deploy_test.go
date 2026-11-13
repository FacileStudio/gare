package engine

import (
	"testing"
	"time"

	"github.com/FacileStudio/gare/apps/api/internal/storage"
)

func TestDeployWaitsForTheAppLock(t *testing.T) {
	store := storage.Open(t.TempDir())
	if err := store.Save(&storage.AppConfig{Name: "app", Branch: "main"}); err != nil {
		t.Fatal(err)
	}
	unlock, err := store.Lock(t.Context(), "app")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	eng, sink := newTestEngine(store, Settings{})
	go func() {
		_, err := eng.Deploy(t.Context(), "app")
		done <- err
	}()
	select {
	case err := <-done:
		t.Fatalf("a deploy ran while another held the app lock: %v", err)
	case <-time.After(400 * time.Millisecond):
	}
	if !sink.hasEvent("step: Waiting for another gare operation on app to finish...") {
		t.Errorf("a deploy waiting for the lock must say so, got %v", sink.recorded())
	}
	unlock()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("the deploy did not proceed once the lock was released")
	}
}
