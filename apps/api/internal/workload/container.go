package workload

import (
	"context"
	"fmt"

	"github.com/FacileStudio/gare/apps/api/internal/podman"
)

// container is the workload type gare builds into a local image and runs as a pod through
// `podman kube play`.
type container struct{}

// ImageRef is the tag gare builds and runs an application image under. The localhost/ prefix
// marks it as locally built, which is what tells podman never to resolve it from a registry.
func ImageRef(name string) string {
	return fmt.Sprintf("localhost/%s:latest", name)
}

// Settle fills what only the checkout can tell a container workload: the port its Containerfile
// exposes and, at a deploy of an app stored without one, a free host port.
func (container) Settle(app App, stage Stage) error {
	cfg := app.Config
	warnStrayComposeFile(app)
	if stage == Deploy && cfg.Port == 0 {
		port, err := app.Store.AvailablePort(0)
		if err != nil {
			return fmt.Errorf("failed to discover free port: %w", err)
		}
		cfg.Port = port
	}
	if cfg.ContainerPort == 0 {
		if exposed := podman.DetectExposedPort(app.RepoDir(), cfg.Containerfile); exposed > 0 {
			cfg.ContainerPort = exposed
		}
	}
	return nil
}

// Prepare writes the pod manifest and the unit that plays it. A deploy first runs the build
// command, builds the image and syncs the manifest from the repository; a create adopts the
// repository's manifest or generates one.
func (container) Prepare(ctx context.Context, app App, stage Stage) error {
	if stage == Create {
		if err := resolveManifest(app); err != nil {
			return err
		}
		return writeContainerUnit(ctx, app)
	}
	if err := runBuildCommand(ctx, app); err != nil {
		return err
	}
	if err := buildImage(ctx, app); err != nil {
		return err
	}
	if err := syncManifest(app); err != nil {
		return err
	}
	return writeContainerUnit(ctx, app)
}

// Teardown removes the local image a container workload was built from, so a destroy does not leave
// a versionless image behind for the next build to reuse silently.
func (container) Teardown(ctx context.Context, app App, _ bool) {
	imageName := ImageRef(app.Config.Name)
	app.Report.Step(fmt.Sprintf("Removing container image %s...", imageName))
	bestEffort(app, "removing container image", func() error { return podman.RemoveImage(ctx, imageName) })
}

// Env returns the pod manifest, which is where a container workload reads its variables.
func (container) Env() EnvStore {
	return podManifestStore()
}

// CheckStart fails a start before systemd does when the image is not built yet, so the operator is
// pointed at the command that builds it instead of a restart loop whose journal only shows podman
// trying to reach a registry named localhost. A podman that cannot be queried is left for systemd
// to report, rather than turning an inspection failure into a start failure of its own.
func (container) CheckStart(ctx context.Context, app App) error {
	name := app.Config.Name
	ref := ImageRef(name)
	exists, err := podman.ImageExists(ctx, ref)
	if err != nil || exists {
		return nil
	}
	return fmt.Errorf("container image %s is not built yet — run `gare deploy %s` to build and start it", ref, name)
}

// CheckStopped has nothing to check: the pod goes down with its unit's `kube down`.
func (container) CheckStopped(context.Context, App) {}
