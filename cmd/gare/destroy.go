package main

import (
	"context"
	"fmt"
	"time"

	"github.com/FacileStudio/gare/internal/appname"
	"github.com/FacileStudio/gare/internal/caddy"
	"github.com/FacileStudio/gare/internal/podman"
	"github.com/FacileStudio/gare/internal/storage"
	"github.com/FacileStudio/gare/internal/systemd"
	"github.com/spf13/cobra"
)

// NewDestroyCmd builds the destroy command.
func NewDestroyCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "destroy <name>",
		Aliases: []string{"delete", "rm"},
		Short:   "Stop and tear down an application and its resources",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if err := appname.Validate(name); err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 2*time.Minute)
			defer cancel()

			appDir := storage.GetAppDir(storage.DefaultBaseDir(), name)
			cfg, workload, err := resolveDestroyWorkload(appDir)
			if err != nil {
				printWarning(fmt.Sprintf("%v — tearing down only what does not depend on the workload type, so a workload may keep running and its image survives: check them with `podman ps` and `podman images`", err))
			}

			teardownServices(ctx, name)
			removeArtifacts(ctx, name, appDir, workload, cfg)
			reloadDaemons(ctx)

			printSuccess(fmt.Sprintf("App %s completely destroyed", name))
			return nil
		},
	}
}

// bestEffort runs one teardown step and reports its failure as a warning, because a destroy keeps
// going through whatever is already gone rather than abandoning the rest of the teardown.
func bestEffort(label string, action func() error) {
	if err := action(); err != nil {
		printWarning(fmt.Sprintf("%s returned error: %v", label, err))
	}
}

func teardownServices(ctx context.Context, name string) {
	printInfo(fmt.Sprintf("Stopping service %s...", name))
	bestEffort("systemctl stop", func() error { return systemd.Stop(ctx, name) })

	printInfo(fmt.Sprintf("Disabling service %s...", name))
	bestEffort("systemctl disable", func() error { return systemd.Disable(ctx, name) })

	printInfo("Removing systemd unit file and enable link...")
	bestEffort("removing unit file", func() error { return systemd.RemoveUnit(name) })
	bestEffort("removing stale Quadlet sources", func() error { return systemd.RemoveLegacyQuadletSources(name) })
}

// resolveDestroyWorkload resolves the workload type an application was deployed as, before anything
// is torn down. destroy branches on this explicit type rather than on the IsStatic and IsCompose
// predicates, which read a configuration gare could not load as a container workload — the reading
// that once removed a compose stack's unit and storage while leaving the stack itself running.
func resolveDestroyWorkload(appDir string) (*storage.AppConfig, storage.WorkloadType, error) {
	cfg, err := storage.LoadConfig(appDir)
	if err != nil {
		return nil, storage.WorkloadUnknown, fmt.Errorf("could not read %s: %w", storage.GetConfigPath(appDir), err)
	}
	workload, err := cfg.ResolveWorkloadType()
	if err != nil {
		return nil, storage.WorkloadUnknown, fmt.Errorf("could not read the workload type from %s: %w", storage.GetConfigPath(appDir), err)
	}
	return cfg, workload, nil
}

func removeArtifacts(ctx context.Context, name, appDir string, workload storage.WorkloadType, cfg *storage.AppConfig) {
	printInfo("Removing Caddy snippet...")
	bestEffort("removing caddy snippet", func() error { return caddy.RemoveSnippet(caddy.ResolveConfDir(), name) })

	teardownWorkload(ctx, name, appDir, workload, cfg)

	printInfo(fmt.Sprintf("Removing app storage at %s...", appDir))
	bestEffort("removing storage", func() error { return storage.DeleteAppStorage(appDir) })
}

// teardownWorkload tears down the workload itself, the one part of a destroy that depends on the
// workload type. Only a compose stack and a container image need it: a static site's content is the
// storage removed either way, and an unknown type is left alone because gare does not know which
// workload it started, so a guessed teardown would be worse than none.
func teardownWorkload(ctx context.Context, name, appDir string, workload storage.WorkloadType, cfg *storage.AppConfig) {
	switch workload {
	case storage.WorkloadCompose:
		destroyComposeWorkload(ctx, appDir, cfg)
	case storage.WorkloadContainer:
		removeContainerImage(ctx, name)
	}
}

// removeContainerImage removes the local image a container workload was built from, so a destroy
// does not leave a versionless image behind for the next build to reuse silently.
func removeContainerImage(ctx context.Context, name string) {
	imageName := fmt.Sprintf("localhost/%s:latest", name)
	printInfo(fmt.Sprintf("Removing container image %s...", imageName))
	bestEffort("removing container image", func() error { return podman.RemoveImage(ctx, imageName) })
}

func reloadDaemons(ctx context.Context) {
	printInfo("Reloading systemd daemon...")
	bestEffort("daemon-reload", func() error { return systemd.DaemonReload(ctx) })

	printInfo("Reloading Caddy...")
	bestEffort("caddy reload", func() error { return caddy.Reload(ctx) })
}
