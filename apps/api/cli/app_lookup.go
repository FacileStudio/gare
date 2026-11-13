package cli

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/FacileStudio/gare/apps/api/internal/appname"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
)

// lockQuietWait is how long a command waits for an app's lock before it says it is waiting, so an
// uncontended lock stays silent and a contended one never looks like a hang.
const lockQuietWait = 200 * time.Millisecond

// loadAppConfig resolves a managed application's storage directory and configuration, so every
// command that operates on one application agrees on validation and on the error an unknown app
// reports.
func loadAppConfig(name string) (string, *storage.AppConfig, error) {
	if err := appname.Validate(name); err != nil {
		return "", nil, err
	}
	store := storage.Default()
	cfg, err := store.Load(name)
	if err != nil {
		return "", nil, appConfigError(name, err)
	}
	return store.AppDir(name), cfg, nil
}

// lockApp takes the lock of an existing app. It reports an unknown app before creating any lock
// file for it; the caller loads the configuration once it holds the lock, so it starts from what
// the previous holder saved rather than from a copy read while it waited.
func lockApp(ctx context.Context, name string) (func(), error) {
	if _, _, err := loadAppConfig(name); err != nil {
		return nil, err
	}
	return waitForAppLock(ctx, name)
}

// waitForAppLock takes an app's lock, and tells the operator when another gare command, or a
// webhook deploy, holds it and this one is waiting.
func waitForAppLock(ctx context.Context, name string) (func(), error) {
	store := storage.Default()
	quick, cancel := context.WithTimeout(ctx, lockQuietWait)
	unlock, err := store.Lock(quick, name)
	cancel()
	if err == nil || !errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
		return unlock, err
	}
	printInfo(fmt.Sprintf("Waiting for another gare operation on %s to finish...", name))
	return store.Lock(ctx, name)
}
