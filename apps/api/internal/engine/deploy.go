package engine

import (
	"context"
	"fmt"
	"strings"

	"github.com/FacileStudio/gare/apps/api/internal/deploys"
	"github.com/FacileStudio/gare/apps/api/internal/git"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
	"github.com/FacileStudio/gare/apps/api/internal/systemd"
	"github.com/FacileStudio/gare/apps/api/internal/workload"
)

// DeployResult describes a finished deploy: what was deployed, at which commit, and where it is
// served.
type DeployResult struct {
	Name    string
	AppType string
	Commit  string
	Target  string
}

// Deploy runs the deployment pipeline for a single application. Every workload type runs the same
// tail: prepare its artifacts, sync ingress, restart the unit, verify health, then clean up. It
// holds the app's lock from start to finish, so a CLI deploy and a webhook deploy of one app never
// interleave their git sync, unit writes and config saves. Each deploy is recorded with its log
// from the moment it starts, before it waits for the lock, so the history shows it at once.
func (e *Engine) Deploy(ctx context.Context, name string, opts DeployOptions) (*DeployResult, error) {
	run, rec, err := e.startRecord(name, deploys.Request{Trigger: opts.Trigger}, opts.QueuedID)
	if err != nil {
		return nil, err
	}
	result, err := run.deploy(ctx, name, rec)
	e.finishRecord(name, rec, err)
	return result, err
}

// deploy runs one recorded deploy under the app's lock.
func (e *Engine) deploy(ctx context.Context, name string, rec *deploys.Recorder) (*DeployResult, error) {
	unlock, cfg, err := e.lockAndLoad(ctx, name)
	if err != nil {
		return nil, err
	}
	defer unlock()
	repoDir := storage.GetRepoDir(e.store.AppDir(name))
	w, err := e.fetchRevision(ctx, repoDir, cfg)
	if err != nil {
		return nil, err
	}
	commit := DeployedCommit(ctx, repoDir)
	e.recordCommit(name, rec, commit)
	e.report(name).Verbose(fmt.Sprintf("Resolved %s app %s: repo %s, port %d, container port %d, unit %s",
		cfg.AppType, name, repoDir, cfg.Port, cfg.ContainerPort, systemd.GetUnitPath(name)))
	if err := w.Prepare(ctx, e.workloadApp(cfg), workload.Deploy); err != nil {
		return nil, err
	}
	if err := e.activate(ctx, cfg); err != nil {
		return nil, err
	}
	return &DeployResult{Name: name, AppType: cfg.AppType, Commit: commit, Target: deployTarget(cfg)}, nil
}

// recordCommit stores the commit a deploy runs in its record, once the checkout knows it.
func (e *Engine) recordCommit(name string, rec *deploys.Recorder, commit string) {
	if commit == "-" {
		return
	}
	if err := rec.SetCommit(commit); err != nil {
		e.report(name).Warn(fmt.Sprintf("Could not record the deployed commit (%v)", err))
	}
}

// fetchRevision aligns the checkout with its remote, then settles the configuration the synced
// gare.yml asks for and records it. It returns the workload the settled configuration runs as,
// which a gare.yml type change may have moved.
func (e *Engine) fetchRevision(ctx context.Context, repoDir string, cfg *storage.AppConfig) (workload.Workload, error) {
	e.report(cfg.Name).Step(fmt.Sprintf("Syncing %s with its git remote...", cfg.Name))
	stdout, stderr := e.sink.Output(cfg.Name)
	opts := git.SyncOptions{RepoDir: repoDir, Branch: cfg.Branch, Auth: e.settings.Auth, Stdout: stdout, Stderr: stderr}
	if err := git.Sync(ctx, opts); err != nil {
		return nil, fmt.Errorf("git sync failed: %w", err)
	}
	if err := resolveDeployConfig(e.store, repoDir, cfg); err != nil {
		return nil, err
	}
	w, err := workload.For(cfg)
	if err != nil {
		return nil, err
	}
	if err := w.Settle(e.workloadApp(cfg), workload.Deploy); err != nil {
		return nil, err
	}
	if err := e.store.Save(cfg); err != nil {
		e.report(cfg.Name).Warn(fmt.Sprintf("Could not persist updated config (%v)", err))
	}
	return w, nil
}

// activate syncs ingress, restarts the unit, and only reports success once the ingress serves the
// app's domains and the health probes pass.
func (e *Engine) activate(ctx context.Context, cfg *storage.AppConfig) error {
	if err := e.activateIngress(ctx, cfg); err != nil {
		return err
	}
	e.report(cfg.Name).Verbose(fmt.Sprintf("Enabling and restarting %s.service", cfg.Name))
	if err := e.restartUnit(ctx, cfg.Name); err != nil {
		return err
	}
	if err := e.verifyHealth(ctx, cfg); err != nil {
		return err
	}
	e.cleanupDeploy(ctx, cfg.Name)
	return nil
}

// deployTarget names where a deployed app is served: its domains and port, or its port alone.
func deployTarget(cfg *storage.AppConfig) string {
	if len(cfg.Domains) > 0 {
		return fmt.Sprintf("%s (:%d)", strings.Join(cfg.Domains, ", "), cfg.Port)
	}
	return fmt.Sprintf("port %d", cfg.Port)
}

// DeployedCommit returns the commit an app's checkout is at, or "-" when it cannot be read.
func DeployedCommit(ctx context.Context, repoDir string) string {
	commit, _ := git.GetCommitHash(ctx, repoDir)
	if commit == "" {
		return "-"
	}
	return commit
}
