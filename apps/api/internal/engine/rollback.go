package engine

import (
	"context"
	"errors"
	"fmt"

	"github.com/FacileStudio/gare/apps/api/internal/deploys"
	"github.com/FacileStudio/gare/apps/api/internal/podman"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
	"github.com/FacileStudio/gare/apps/api/internal/workload"
)

// RolledBackError is the error of a deploy that failed its health probes and was replaced by an
// earlier deploy that passes them: the app is up, on the earlier build, and the new one failed.
type RolledBackError struct {
	Cause error
	To    string
}

// Error says the deploy failed and which deploy runs instead.
func (e *RolledBackError) Error() string {
	return fmt.Sprintf("%v; rolled back to deploy %s", e.Cause, e.To)
}

// Unwrap returns the failure that caused the rollback.
func (e *RolledBackError) Unwrap() error {
	return e.Cause
}

// asRolledBack reports whether err is a deploy that rolled back, and to where.
func asRolledBack(err error) (*RolledBackError, bool) {
	var rolled *RolledBackError
	ok := errors.As(err, &rolled)
	return rolled, ok
}

// rollBackFailed answers a failed health check after the restart. A container app goes back to the
// newest successful deploy of another commit whose image is kept, and the error says so; any other
// app, or one with nothing to go back to, fails with cause as before.
func (e *Engine) rollBackFailed(ctx context.Context, cfg *storage.AppConfig, commit string, cause error) error {
	if !workload.CanRollBack(cfg) {
		return cause
	}
	target, ok := e.rollbackTarget(ctx, cfg.Name, commit)
	if !ok {
		return cause
	}
	report := e.report(cfg.Name)
	report.Warn(fmt.Sprintf("%v; rolling back to deploy %s (%s)", cause, target.ID, target.Commit))
	if err := e.restore(ctx, cfg, target.Commit); err != nil {
		return fmt.Errorf("%w; rolling back to deploy %s failed too: %w", cause, target.ID, err)
	}
	report.Done(fmt.Sprintf("Rolled back %s to deploy %s (%s)", cfg.Name, target.ID, target.Commit))
	return &RolledBackError{Cause: cause, To: target.ID}
}

// restore runs the image an earlier deploy of commit built and verifies it is healthy.
func (e *Engine) restore(ctx context.Context, cfg *storage.AppConfig, commit string) error {
	if err := workload.PointImage(ctx, e.workloadApp(cfg), commit); err != nil {
		return err
	}
	if err := e.restartUnit(ctx, cfg.Name); err != nil {
		return err
	}
	return e.verifyHealth(ctx, cfg)
}

// rollbackTarget returns the newest successful deploy of a commit other than skip whose image is
// still kept, the deploy a rollback goes back to.
func (e *Engine) rollbackTarget(ctx context.Context, name, skip string) (deploys.Deploy, bool) {
	list, err := deploys.Open(e.store.AppDir(name), name).List()
	if err != nil {
		return deploys.Deploy{}, false
	}
	for _, d := range list {
		if d.Status != deploys.StatusSucceeded || d.Commit == "" || deploys.ShortSHA(d.Commit) == deploys.ShortSHA(skip) {
			continue
		}
		if imageKept(ctx, name, d.Commit) {
			return d, true
		}
	}
	return deploys.Deploy{}, false
}

// imageKept reports whether the image a deploy of commit built is still in local storage.
func imageKept(ctx context.Context, name, commit string) bool {
	exists, err := podman.ImageExists(ctx, workload.ImageTag(name, commit))
	return err == nil && exists
}
