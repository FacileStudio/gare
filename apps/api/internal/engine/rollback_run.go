package engine

import (
	"context"
	"fmt"

	"github.com/FacileStudio/gare/apps/api/internal/deploys"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
	"github.com/FacileStudio/gare/apps/api/internal/workload"
)

// RollbackOptions says who asked for a rollback and, optionally, which deploy to go back to. With
// no deploy id it goes back to the newest successful deploy of another commit than the current one.
type RollbackOptions struct {
	Trigger deploys.Trigger
	To      string
}

// Rollback makes a container app run the image an earlier successful deploy built, under the app's
// lock, and records it as a deploy of its own whose rolled_back_to names that deploy. Static and
// compose apps keep no earlier image, so they are refused before anything changes.
func (e *Engine) Rollback(ctx context.Context, name string, opts RollbackOptions) (*DeployResult, error) {
	cfg, err := e.Config(name)
	if err != nil {
		return nil, err
	}
	if !workload.CanRollBack(cfg) {
		return nil, fmt.Errorf("%s is a %s app: rollback only supports container workloads, "+
			"the only type gare keeps earlier images of; redeploy an earlier commit instead", name, cfg.AppType)
	}
	unlock, cfg, err := e.lockAndLoad(ctx, name)
	if err != nil {
		return nil, err
	}
	defer unlock()
	target, err := e.resolveRollbackTarget(ctx, name, opts.To)
	if err != nil {
		return nil, err
	}
	return e.rollBackTo(ctx, cfg, target, opts.Trigger)
}

// rollBackTo records and runs one rollback to target.
func (e *Engine) rollBackTo(ctx context.Context, cfg *storage.AppConfig, target deploys.Deploy,
	trigger deploys.Trigger) (*DeployResult, error) {
	req := deploys.Request{Trigger: trigger, Commit: target.Commit, RolledBackTo: target.ID}
	run, rec, err := e.startRecord(cfg.Name, req, "")
	if err != nil {
		return nil, err
	}
	run.report(cfg.Name).Step(fmt.Sprintf("Rolling back %s to deploy %s (%s)...", cfg.Name, target.ID, target.Commit))
	err = run.restore(ctx, cfg, target.Commit)
	e.finishRecord(cfg.Name, rec, err)
	if err != nil {
		return nil, err
	}
	return &DeployResult{Name: cfg.Name, AppType: cfg.AppType, Commit: target.Commit, Target: deployTarget(cfg)}, nil
}

// resolveRollbackTarget returns the deploy a rollback goes back to: the one named by id, which must
// have succeeded and still have its image, or else the newest successful deploy of another commit
// than the one running now.
func (e *Engine) resolveRollbackTarget(ctx context.Context, name, id string) (deploys.Deploy, error) {
	if id == "" {
		target, ok := e.rollbackTarget(ctx, name, e.currentCommit(name))
		if !ok {
			return deploys.Deploy{}, fmt.Errorf("%s has no earlier successful deploy whose image is kept", name)
		}
		return target, nil
	}
	target, err := deploys.Open(e.store.AppDir(name), name).Get(id)
	if err != nil {
		return deploys.Deploy{}, err
	}
	if target.Status != deploys.StatusSucceeded || target.Commit == "" {
		return deploys.Deploy{}, fmt.Errorf("deploy %s is %s: only a successful deploy can be rolled back to", id, target.Status)
	}
	if !imageKept(ctx, name, target.Commit) {
		return deploys.Deploy{}, fmt.Errorf("the image of deploy %s (%s) is no longer kept: gare keeps the last %d",
			id, target.Commit, keptImageTags)
	}
	return target, nil
}

// currentCommit returns the commit of the newest successful deploy, the build the app runs now.
func (e *Engine) currentCommit(name string) string {
	list, err := deploys.Open(e.store.AppDir(name), name).List()
	if err != nil {
		return ""
	}
	for _, d := range list {
		if d.Status == deploys.StatusSucceeded {
			return d.Commit
		}
	}
	return ""
}

// CanRollback reports whether an app can be rolled back now: its workload type keeps earlier images
// and an earlier successful deploy's image is still there. The API shows it as App.rollback.
func (e *Engine) CanRollback(ctx context.Context, name string) bool {
	cfg, err := e.Config(name)
	if err != nil || !workload.CanRollBack(cfg) {
		return false
	}
	_, ok := e.rollbackTarget(ctx, name, e.currentCommit(name))
	return ok
}
