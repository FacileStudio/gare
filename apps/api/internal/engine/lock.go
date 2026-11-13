package engine

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/FacileStudio/gare/apps/api/internal/appname"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
	"github.com/FacileStudio/gare/apps/api/internal/workload"
)

// lockQuietWait is how long an operation waits for an app's lock before it says it is waiting, so
// an uncontended lock stays silent and a contended one never looks like a hang.
const lockQuietWait = 200 * time.Millisecond

// Config resolves a managed application's configuration, so every operation on one application
// agrees on validation and on the error an unknown app reports.
func (e *Engine) Config(name string) (*storage.AppConfig, error) {
	if err := appname.Validate(name); err != nil {
		return nil, err
	}
	cfg, err := e.store.Load(name)
	if err != nil {
		return nil, appConfigError(name, err)
	}
	return cfg, nil
}

// appConfigError wraps a failed application config lookup with a remedy.
func appConfigError(name string, err error) error {
	return fmt.Errorf("app %q not found — run `gare list` to see managed apps: %w", name, err)
}

// lockAndLoad takes the lock of an existing app and loads its configuration. It reports an unknown
// app before creating any lock file for it, and loads the configuration only once it holds the
// lock, so it starts from what the previous holder saved rather than from a copy read while it
// waited. A container app an older gare left with its variables in the manifest has them moved to
// its env file here, under the lock and from the type it was stored with, so a deploy that changes
// its workload type still carries them over.
func (e *Engine) lockAndLoad(ctx context.Context, name string) (func(), *storage.AppConfig, error) {
	if _, err := e.Config(name); err != nil {
		return nil, nil, err
	}
	unlock, err := e.waitForLock(ctx, name)
	if err != nil {
		return nil, nil, err
	}
	cfg, err := e.Config(name)
	if err == nil {
		err = workload.MigrateEnv(e.workloadApp(cfg))
	}
	if err != nil {
		unlock()
		return nil, nil, err
	}
	return unlock, cfg, nil
}

// waitForLock takes an app's lock, and says so when another gare operation, or a webhook deploy,
// holds it and this one is waiting.
func (e *Engine) waitForLock(ctx context.Context, name string) (func(), error) {
	quick, cancel := context.WithTimeout(ctx, lockQuietWait)
	unlock, err := e.store.Lock(quick, name)
	cancel()
	if err == nil || !errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
		return unlock, err
	}
	e.report(name).Step(fmt.Sprintf("Waiting for another gare operation on %s to finish...", name))
	return e.store.Lock(ctx, name)
}
