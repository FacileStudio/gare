package engine

import (
	"context"
	"fmt"

	"github.com/FacileStudio/gare/apps/api/internal/systemd"
	"github.com/FacileStudio/gare/apps/api/internal/workload"
)

// EnvResult says whether an environment change restarted the app, and the state systemd reported
// for a unit that was not running and so kept its old environment until its next start.
type EnvResult struct {
	Restarted bool
	State     string
}

// envChange is one environment edit: the verb a failure names, the edit itself, and the line that
// reports it once the store it landed in is known.
type envChange struct {
	action  string
	apply   func(store workload.EnvStore, appDir string) error
	summary func(label string) string
}

// Env returns an app's environment variables from the store its workload reads.
func (e *Engine) Env(name string) (map[string]string, error) {
	cfg, err := e.Config(name)
	if err != nil {
		return nil, err
	}
	w, err := workload.For(cfg)
	if err != nil {
		return nil, err
	}
	store := w.Env()
	envs, err := store.Read(e.store.AppDir(name))
	if err != nil {
		return nil, fmt.Errorf("failed to read environment variables from the %s: %w", store.Label, err)
	}
	return envs, nil
}

// SetEnv sets environment variables and restarts the app when it is running.
func (e *Engine) SetEnv(ctx context.Context, name string, vars map[string]string) (*EnvResult, error) {
	return e.changeEnv(ctx, name, envChange{
		action: "set",
		apply:  func(store workload.EnvStore, appDir string) error { return store.Write(appDir, vars) },
		summary: func(label string) string {
			return fmt.Sprintf("Updated %d environment variable(s) in %s", len(vars), label)
		},
	})
}

// UnsetEnv removes environment variables and restarts the app when it is running.
func (e *Engine) UnsetEnv(ctx context.Context, name string, keys []string) (*EnvResult, error) {
	return e.changeEnv(ctx, name, envChange{
		action: "unset",
		apply:  func(store workload.EnvStore, appDir string) error { return store.Remove(appDir, keys) },
		summary: func(label string) string {
			return fmt.Sprintf("Removed %d environment variable(s) from %s", len(keys), label)
		},
	})
}

// LoadEnv sets the variables read from source, an env file, and restarts the app when it is
// running.
func (e *Engine) LoadEnv(ctx context.Context, name string, vars map[string]string, source string) (*EnvResult, error) {
	return e.changeEnv(ctx, name, envChange{
		action: "load",
		apply:  func(store workload.EnvStore, appDir string) error { return store.Write(appDir, vars) },
		summary: func(string) string {
			return fmt.Sprintf("Loaded %d environment variable(s) from %s", len(vars), source)
		},
	})
}

// changeEnv applies one edit under the app's lock, in the store its workload reads, so the change
// lands where the workload looks, then restarts a running app so it takes effect.
func (e *Engine) changeEnv(ctx context.Context, name string, change envChange) (*EnvResult, error) {
	unlock, cfg, err := e.lockAndLoad(ctx, name)
	if err != nil {
		return nil, err
	}
	defer unlock()
	w, err := workload.For(cfg)
	if err != nil {
		return nil, err
	}
	store := w.Env()
	if err := change.apply(store, e.store.AppDir(name)); err != nil {
		return nil, fmt.Errorf("failed to %s environment variables in the %s: %w", change.action, store.Label, err)
	}
	e.report(name).Done(change.summary(store.Label))
	return e.restartIfActive(ctx, name)
}

// restartIfActive restarts a running app so it reads its new environment, and leaves a stopped one
// alone.
func (e *Engine) restartIfActive(ctx context.Context, name string) (*EnvResult, error) {
	status, err := systemd.IsActive(ctx, name)
	if err != nil || status != "active" {
		return &EnvResult{State: status}, nil
	}
	e.report(name).Step(fmt.Sprintf("Restarting %s.service to apply environment changes...", name))
	if err := systemd.Restart(ctx, name); err != nil {
		return nil, fmt.Errorf("failed to restart %s: %w", name, err)
	}
	if err := systemd.WaitForState(ctx, name, "active"); err != nil {
		return nil, fmt.Errorf("failed to verify %s active state: %w", name, err)
	}
	return &EnvResult{Restarted: true, State: "active"}, nil
}
