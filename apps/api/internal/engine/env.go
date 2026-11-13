package engine

import (
	"context"
	"fmt"

	"github.com/FacileStudio/gare/apps/api/internal/storage"
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
// reports it.
type envChange struct {
	action  string
	apply   func(appDir string) error
	summary string
}

// Env returns an app's environment variables from its env file.
func (e *Engine) Env(name string) (map[string]string, error) {
	cfg, err := e.Config(name)
	if err != nil {
		return nil, err
	}
	envs, err := workload.ReadEnv(e.workloadApp(cfg))
	if err != nil {
		return nil, fmt.Errorf("failed to read environment variables from the app env file: %w", err)
	}
	return envs, nil
}

// SetEnv sets environment variables and restarts the app when it is running.
func (e *Engine) SetEnv(ctx context.Context, name string, vars map[string]string) (*EnvResult, error) {
	return e.changeEnv(ctx, name, envChange{
		action:  "set",
		apply:   func(appDir string) error { return storage.SetAppEnv(appDir, vars) },
		summary: fmt.Sprintf("Updated %d environment variable(s) in app env file", len(vars)),
	})
}

// UnsetEnv removes environment variables and restarts the app when it is running.
func (e *Engine) UnsetEnv(ctx context.Context, name string, keys []string) (*EnvResult, error) {
	return e.changeEnv(ctx, name, envChange{
		action:  "unset",
		apply:   func(appDir string) error { return storage.UnsetAppEnv(appDir, keys) },
		summary: fmt.Sprintf("Removed %d environment variable(s) from app env file", len(keys)),
	})
}

// LoadEnv sets the variables read from source, an env file, and restarts the app when it is
// running.
func (e *Engine) LoadEnv(ctx context.Context, name string, vars map[string]string, source string) (*EnvResult, error) {
	return e.changeEnv(ctx, name, envChange{
		action:  "load",
		apply:   func(appDir string) error { return storage.SetAppEnv(appDir, vars) },
		summary: fmt.Sprintf("Loaded %d environment variable(s) from %s", len(vars), source),
	})
}

// changeEnv applies one edit to the app env file under the app's lock, has the workload render it
// into what its runtime reads, then restarts a running app so it takes effect. lockAndLoad has
// already moved any variables an older gare kept in a container's manifest into the env file.
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
	app := e.workloadApp(cfg)
	if err := change.apply(app.Dir); err != nil {
		return nil, fmt.Errorf("failed to %s environment variables in the app env file: %w", change.action, err)
	}
	if err := w.RenderEnv(app); err != nil {
		return nil, err
	}
	e.report(name).Done(change.summary)
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
