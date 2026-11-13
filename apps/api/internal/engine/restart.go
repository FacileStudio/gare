package engine

import (
	"context"
	"fmt"

	"github.com/FacileStudio/gare/apps/api/internal/podman"
	"github.com/FacileStudio/gare/apps/api/internal/systemd"
)

// unitFailureLogLines is how much of a unit's journal a failed restart shows.
const unitFailureLogLines = 10

// restartUnit reloads systemd, enables the app's unit and restarts it. This is the daemon-reload
// that makes systemd load a rewritten unit, so it runs only after the workload adapter retired the
// workload it replaces.
func (e *Engine) restartUnit(ctx context.Context, name string) error {
	if err := systemd.DaemonReload(ctx); err != nil {
		return fmt.Errorf("systemctl daemon-reload failed: %w", err)
	}
	if err := systemd.Enable(ctx, name); err != nil {
		return fmt.Errorf("failed to enable service: %w", err)
	}
	if err := systemd.Restart(ctx, name); err != nil {
		e.showUnitFailure(ctx, name)
		return fmt.Errorf("failed to restart service: %w", err)
	}
	return nil
}

// showUnitFailure shows the unit's own journal, because systemd only reports that the control
// process exited while the reason a workload refused to start is in its logs.
func (e *Engine) showUnitFailure(ctx context.Context, name string) {
	logs := systemd.RecentLogs(ctx, name, unitFailureLogLines)
	if logs == "" {
		return
	}
	_, stderr := e.sink.Output(name)
	fmt.Fprintln(stderr, logs)
}

// cleanupDeploy runs the best-effort cleanup after a deploy. Pruning is a nicety rather than a
// deployment step, so its output is shown only in verbose mode: podman exits non-zero when a
// leftover buildah working container holds a dangling image, which says nothing about the deploy.
func (e *Engine) cleanupDeploy(ctx context.Context, name string) {
	report := e.report(name)
	output, err := podman.PruneImages(ctx)
	if err != nil {
		report.Verbose(fmt.Sprintf("Skipped image pruning: %v (%s)", err, output))
		return
	}
	if output != "" {
		report.Verbose(fmt.Sprintf("Image pruning: %s", output))
	}
}
