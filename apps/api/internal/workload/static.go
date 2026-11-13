package workload

import (
	"context"
	"fmt"

	"github.com/FacileStudio/gare/apps/api/internal/ingress"
	"github.com/FacileStudio/gare/apps/api/internal/repopath"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
	"github.com/FacileStudio/gare/apps/api/internal/systemd"
)

// static is the workload type that serves a directory of the repository through the bundled Caddy
// container.
type static struct{}

// Settle has nothing to read from the checkout for a static site beyond warning about a compose
// file it ignores.
func (static) Settle(app App, _ Stage) error {
	warnStrayComposeFile(app)
	return nil
}

// Prepare writes the unit serving the static directory, after the build command at a deploy. The
// directory is resolved inside the repository, so a symlink cannot publish files outside it.
func (static) Prepare(ctx context.Context, app App, stage Stage) error {
	if stage == Deploy {
		if err := runBuildCommand(ctx, app); err != nil {
			return err
		}
	}
	staticPath, err := repopath.Join("static_dir", app.RepoDir(), app.Config.StaticDir)
	if err != nil {
		return err
	}
	return writeStaticUnit(ctx, app, staticPath)
}

// Teardown has nothing to remove: a static site's content is the storage a destroy removes anyway.
func (static) Teardown(context.Context, App, bool) {}

// RenderEnv rewrites the variable list the static container's podman reads its values through.
func (static) RenderEnv(app App) error {
	return writePodmanEnv(app)
}

// CheckStart has nothing to check: the Caddy image is pulled by the unit itself.
func (static) CheckStart(context.Context, App) error {
	return nil
}

// CheckStopped has nothing to check: the unit removes the container it started.
func (static) CheckStopped(context.Context, App) {}

// writeStaticUnit writes the systemd unit running the application static directory, with the
// Caddyfile it serves and the env file it loads.
func writeStaticUnit(ctx context.Context, app App, rootDir string) error {
	name := app.Config.Name
	previous, err := claimUnit(ctx, name, true)
	if err != nil {
		return err
	}
	if err := storage.EnsureAppEnvFile(app.Dir); err != nil {
		return fmt.Errorf("failed to prepare the application env file: %w", err)
	}
	if err := writePodmanEnv(app); err != nil {
		return err
	}
	data := systemd.StaticSiteUnit(name, app.Config.Port, rootDir, storage.GetAppEnvPath(app.Dir),
		storage.GetAppStaticConfigPath(app.Dir))
	if err := ingress.WriteStaticServerConfig(data.ConfigFile, data.RootMount, data.ContainerPort); err != nil {
		return fmt.Errorf("failed to write the static site Caddyfile: %w", err)
	}
	return handOver(ctx, app, systemd.StaticUnitDescription(name), previous,
		func() error { return systemd.WriteStaticUnit(data) })
}
