package engine

import (
	"context"
	"fmt"

	"github.com/FacileStudio/gare/apps/api/internal/caddy"
	"github.com/FacileStudio/gare/apps/api/internal/systemd"
	"github.com/FacileStudio/gare/apps/api/internal/workload"
)

// StopResult is the state systemd reports for a unit once a stop settled.
type StopResult struct {
	ActiveState string
}

// Start starts an app's unit once its workload type agrees it can run and its ingress serves it,
// and waits until systemd reports it active.
func (e *Engine) Start(ctx context.Context, name string) error {
	if err := e.prepareStart(ctx, name); err != nil {
		return err
	}
	if err := systemd.Start(ctx, name); err != nil {
		return fmt.Errorf("failed to start %q: %w", name, err)
	}
	if err := systemd.WaitForState(ctx, name, "active"); err != nil {
		return fmt.Errorf("failed to verify %q active state: %w", name, err)
	}
	return nil
}

// Restart restarts an app's unit under the same checks as Start.
func (e *Engine) Restart(ctx context.Context, name string) error {
	if err := e.prepareStart(ctx, name); err != nil {
		return err
	}
	if err := systemd.Restart(ctx, name); err != nil {
		return fmt.Errorf("failed to restart %q: %w", name, err)
	}
	if err := systemd.WaitForState(ctx, name, "active"); err != nil {
		return fmt.Errorf("failed to verify %q active state: %w", name, err)
	}
	return nil
}

// prepareStart runs the workload type's start check, then activates the app's ingress.
func (e *Engine) prepareStart(ctx context.Context, name string) error {
	cfg, err := e.Config(name)
	if err != nil {
		return err
	}
	w, err := workload.For(cfg)
	if err != nil {
		return err
	}
	if err := w.CheckStart(ctx, e.workloadApp(cfg)); err != nil {
		return err
	}
	return e.activateIngress(ctx, cfg)
}

// Stop takes the app out of the ingress, stops its unit and waits for systemd to settle. A unit
// that stopped with a failure result is reported but does not fail the stop.
func (e *Engine) Stop(ctx context.Context, name string) (*StopResult, error) {
	cfg, err := e.Config(name)
	if err != nil {
		return nil, err
	}
	report := e.report(name)
	if err := caddy.RemoveSnippet(caddy.ResolveConfDir(), name); err == nil {
		if reloadErr := caddy.Reload(ctx); reloadErr != nil {
			report.Warn(fmt.Sprintf("Caddy reload error: %v", reloadErr))
		}
	}
	if err := systemd.Stop(ctx, name); err != nil {
		return nil, fmt.Errorf("failed to stop %q: %w", name, err)
	}
	props, err := systemd.WaitForStop(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to verify %q stopped: %w", name, err)
	}
	if props.Result != "" && props.Result != "success" {
		report.Warn(fmt.Sprintf("Service %s exited with result %s (%s); inspect with: gare logs %s",
			name, props.Result, props.SubState, name))
	}
	if w, err := workload.For(cfg); err == nil {
		w.CheckStopped(ctx, e.workloadApp(cfg))
	}
	return &StopResult{ActiveState: props.ActiveState}, nil
}
