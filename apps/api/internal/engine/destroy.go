package engine

import (
	"context"
	"fmt"

	"github.com/FacileStudio/gare/apps/api/internal/appname"
	"github.com/FacileStudio/gare/apps/api/internal/ingress"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
	"github.com/FacileStudio/gare/apps/api/internal/systemd"
	"github.com/FacileStudio/gare/apps/api/internal/workload"
)

// destroyPlan names everything one destroy acts on, so the teardown steps agree on the app, on the
// workload it runs as, nil when that is unknown, and on whether its data volumes go with it.
type destroyPlan struct {
	name     string
	cfg      *storage.AppConfig
	workload workload.Workload
	volumes  bool
}

// Destroy tears an application down completely: its unit, its ingress, its workload and its
// storage, holding the app's lock. It needs no readable configuration, so a broken app can still be
// removed; then only what does not depend on the workload type is torn down. Confirming the destroy
// is the caller's job.
func (e *Engine) Destroy(ctx context.Context, name string, volumes bool) error {
	if err := appname.Validate(name); err != nil {
		return err
	}
	unlock, err := e.waitForLock(ctx, name)
	if err != nil {
		return err
	}
	defer unlock()
	plan := destroyPlan{name: name, volumes: volumes}
	cfg, workloadType, err := resolveDestroyWorkload(e.store, name)
	if err != nil {
		e.report(name).Warn(fmt.Sprintf("%v — tearing down only what does not depend on the workload type, "+
			"so a workload may keep running and its image survives: check them with `podman ps` and `podman images`", err))
	} else if workloadType != storage.WorkloadUnknown {
		plan.cfg = cfg
		plan.workload, _ = workload.For(cfg)
	}
	e.teardownServices(ctx, name)
	e.removeArtifacts(ctx, plan)
	e.reloadDaemons(ctx, name)
	return nil
}

// resolveDestroyWorkload resolves the workload type an application was deployed as, before anything
// is torn down. destroy acts on this explicit type rather than defaulting a configuration gare could
// not load to a container workload — the reading that once removed a compose stack's unit and
// storage while leaving the stack itself running.
func resolveDestroyWorkload(store *storage.Store, name string) (*storage.AppConfig, storage.WorkloadType, error) {
	configPath := storage.GetConfigPath(store.AppDir(name))
	cfg, err := store.Load(name)
	if err != nil {
		return nil, storage.WorkloadUnknown, fmt.Errorf("could not read %s: %w", configPath, err)
	}
	workloadType, err := cfg.ResolveWorkloadType()
	if err != nil {
		return nil, storage.WorkloadUnknown, fmt.Errorf("could not read the workload type from %s: %w", configPath, err)
	}
	return cfg, workloadType, nil
}

// bestEffort runs one teardown step and reports its failure as a warning, because a destroy keeps
// going through whatever is already gone rather than abandoning the rest of the teardown.
func (e *Engine) bestEffort(name, label string, action func() error) {
	if err := action(); err != nil {
		e.report(name).Warn(fmt.Sprintf("%s returned error: %v", label, err))
	}
}

// teardownServices stops and disables the app's unit, then removes its unit file, enable link and
// any Quadlet source an older gare left behind.
func (e *Engine) teardownServices(ctx context.Context, name string) {
	report := e.report(name)
	report.Step(fmt.Sprintf("Stopping service %s...", name))
	e.bestEffort(name, "systemctl stop", func() error { return systemd.Stop(ctx, name) })

	report.Step(fmt.Sprintf("Disabling service %s...", name))
	e.bestEffort(name, "systemctl disable", func() error { return systemd.Disable(ctx, name) })

	report.Step("Removing systemd unit file and enable link...")
	e.bestEffort(name, "removing unit file", func() error { return systemd.RemoveUnit(name) })
	e.bestEffort(name, "removing stale Quadlet sources", func() error { return systemd.RemoveLegacyQuadletSources(name) })
}

// removeArtifacts removes the app's ingress snippet, its workload when the type is known, and its
// storage. An unknown type is left alone because gare does not know which workload it started, so
// a guessed teardown would be worse than none.
func (e *Engine) removeArtifacts(ctx context.Context, plan destroyPlan) {
	report := e.report(plan.name)
	report.Step("Removing Caddy snippet and reloading Caddy...")
	e.bestEffort(plan.name, "removing caddy snippet", func() error { return ingress.Remove(ctx, plan.name) })

	if plan.workload != nil {
		plan.workload.Teardown(ctx, e.workloadApp(plan.cfg), plan.volumes)
	}

	report.Step(fmt.Sprintf("Removing app storage at %s...", e.store.AppDir(plan.name)))
	e.bestEffort(plan.name, "removing storage", func() error { return e.store.Delete(plan.name) })
}

// reloadDaemons makes systemd forget the removed unit.
func (e *Engine) reloadDaemons(ctx context.Context, name string) {
	report := e.report(name)
	report.Step("Reloading systemd daemon...")
	e.bestEffort(name, "daemon-reload", func() error { return systemd.DaemonReload(ctx) })
}
