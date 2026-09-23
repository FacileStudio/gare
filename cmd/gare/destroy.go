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
			cfg, _ := storage.LoadConfig(appDir)

			teardownServices(ctx, name)
			removeArtifacts(ctx, name, appDir, cfg)
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

func removeArtifacts(ctx context.Context, name, appDir string, cfg *storage.AppConfig) {
	printInfo("Removing Caddy snippet...")
	bestEffort("removing caddy snippet", func() error { return caddy.RemoveSnippet(caddy.ResolveConfDir(), name) })

	if cfg.IsCompose() {
		destroyComposeWorkload(ctx, appDir, cfg)
	} else if !cfg.IsStatic() {
		imageName := fmt.Sprintf("localhost/%s:latest", name)
		printInfo(fmt.Sprintf("Removing container image %s...", imageName))
		bestEffort("removing container image", func() error { return podman.RemoveImage(ctx, imageName) })
	}

	printInfo(fmt.Sprintf("Removing app storage at %s...", appDir))
	bestEffort("removing storage", func() error { return storage.DeleteAppStorage(appDir) })
}

func reloadDaemons(ctx context.Context) {
	printInfo("Reloading systemd daemon...")
	bestEffort("daemon-reload", func() error { return systemd.DaemonReload(ctx) })

	printInfo("Reloading Caddy...")
	bestEffort("caddy reload", func() error { return caddy.Reload(ctx) })
}
