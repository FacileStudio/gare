package workload

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/FacileStudio/gare/apps/api/internal/compose"
	"github.com/FacileStudio/gare/apps/api/internal/health"
	"github.com/FacileStudio/gare/apps/api/internal/podman"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
)

// composeStack is the workload type that runs a repository's own compose file through
// `podman compose`. It is named so it does not shadow the compose package it drives.
type composeStack struct{}

// Settle settles the compose file a new app runs and checks how gare routes to it. At a deploy it
// gives a service-routed app stored without a port a free one and, when gare.yml declares no probes,
// derives them afresh from the compose file.
func (composeStack) Settle(app App, stage Stage) error {
	cfg := app.Config
	if stage == Create {
		composeFile, err := locateCompose(app)
		if err != nil {
			return err
		}
		cfg.ComposeFile = composeFile
		return nil
	}
	if err := assignComposePort(app); err != nil {
		return err
	}
	gf, err := storage.LoadGareFile(app.RepoDir())
	if err != nil {
		return err
	}
	if gf == nil || len(gf.Healthchecks) == 0 {
		cfg.SetHealth(cfg.HealthPath(), deriveComposeProbes(app))
	}
	return nil
}

// Prepare resolves the compose file, checks the routed port or service and, at a deploy, the compose provider
// and the build command, then writes the synthesized unit the stack is started and torn down
// through. A deploy ends by warning when the stack is reachable from this host alone.
func (composeStack) Prepare(ctx context.Context, app App, stage Stage) error {
	cfg := app.Config
	composeFile, err := locateCompose(app)
	if err != nil {
		return err
	}
	if stage == Create {
		cfg.ComposeFile = composeFile
		return writeComposeUnit(ctx, app, composeFile)
	}
	if err := podman.CheckComposeProvider(ctx); err != nil {
		return err
	}
	if err := runBuildCommand(ctx, app); err != nil {
		return err
	}
	if err := writeComposeUnit(ctx, app, composeFile); err != nil {
		return err
	}
	warnComposeReachability(app)
	return nil
}

// Teardown takes the stack down through `podman compose down`, with its named volumes only when the
// operator asked for them, and reports any container that survived.
func (composeStack) Teardown(ctx context.Context, app App, volumes bool) {
	cfg := app.Config
	composeFile, err := compose.LocateFile(app.RepoDir(), cfg.ComposeFile)
	if err != nil {
		app.Report.Warn(fmt.Sprintf("Skipping compose teardown: %v", err))
		return
	}
	app.Report.Step(fmt.Sprintf("Removing compose stack %s%s...", composeFile, volumeNote(volumes)))
	downOpts := podman.ComposeDownOptions{
		RepoDir:      app.RepoDir(),
		ComposeFile:  composeFile,
		OverrideFile: teardownOverride(app),
		Project:      ProjectName(cfg.Name),
		Volumes:      volumes,
		Stdout:       app.Stdout,
		Stderr:       app.Stderr,
	}
	if err := podman.ComposeDown(ctx, downOpts); err != nil {
		app.Report.Warn(fmt.Sprintf("podman compose down returned error: %v", err))
	}
	verifyContainersGone(ctx, app, "teardown")
}

// RenderEnv has nothing to render: the compose unit loads the app env file itself through
// EnvironmentFile=.
func (composeStack) RenderEnv(App) error {
	return nil
}

// CheckStart warns when the stack is reachable from this host alone; it never refuses a start.
func (composeStack) CheckStart(_ context.Context, app App) error {
	warnComposeReachability(app)
	return nil
}

// CheckStopped reports compose containers that outlived the unit's `compose down`.
func (composeStack) CheckStopped(ctx context.Context, app App) {
	verifyContainersGone(ctx, app, "stop")
}

// volumeNote says whether a compose teardown takes the stack's named volumes with it.
func volumeNote(volumes bool) string {
	if volumes {
		return " including its volumes"
	}
	return ", keeping its volumes"
}

// deriveComposeProbes returns one probe per published port whose service healthcheck issues an
// HTTP request, the port gare publishes for the routed service included, or none when the compose
// file implies none.
func deriveComposeProbes(app App) []health.Probe {
	repoDir := app.RepoDir()
	composeFile, err := compose.LocateFile(repoDir, app.Config.ComposeFile)
	if err != nil {
		return nil
	}
	probes, err := compose.HealthProbes(filepath.Join(repoDir, composeFile), routedPublishes(app.Config)...)
	if err != nil || len(probes) == 0 {
		return nil
	}
	app.Report.Step(fmt.Sprintf("Derived %d health probe(s) from %s: %s", len(probes), composeFile, probeSummary(probes)))
	return probes
}
