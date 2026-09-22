package main

import (
	"context"
	"fmt"
	"time"

	"github.com/FacileStudio/gare/internal/builder"
	"github.com/FacileStudio/gare/internal/caddy"
	"github.com/FacileStudio/gare/internal/quadlet"
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
			if err := storage.ValidateAppName(name); err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 2*time.Minute)
			defer cancel()

			baseDir := storage.DefaultBaseDir()
			appDir := storage.GetAppDir(baseDir, name)

			cfg, _ := storage.LoadConfig(appDir)

			teardownServices(ctx, name, cfg)
			removeArtifacts(ctx, name, appDir, cfg)
			reloadDaemons(ctx)

			printSuccess(fmt.Sprintf("App %s completely destroyed", name))
			return nil
		},
	}
}

func teardownServices(ctx context.Context, name string, cfg *storage.AppConfig) {
	stopService(ctx, name)
	if cfg == nil || !cfg.UsesQuadletUnit() {
		disableService(ctx, name)
	}
	removeUnitFiles(name)
}

func stopService(ctx context.Context, name string) {
	printInfo(fmt.Sprintf("Stopping service %s...", name))
	if err := systemd.Stop(ctx, name); err != nil {
		printWarning(fmt.Sprintf("systemctl stop returned error: %v", err))
	}
}

func disableService(ctx context.Context, name string) {
	printInfo(fmt.Sprintf("Disabling service %s...", name))
	if err := systemd.Disable(ctx, name); err != nil {
		printWarning(fmt.Sprintf("systemctl disable returned error: %v", err))
	}
}

func removeUnitFiles(name string) {
	printInfo("Removing systemd unit file and enable link...")
	if err := systemd.RemoveUnit(name); err != nil {
		printWarning(fmt.Sprintf("removing unit file returned error: %v", err))
	}
	if err := quadlet.Remove(name); err != nil {
		printWarning(fmt.Sprintf("removing quadlet unit returned error: %v", err))
	}
}

func removeArtifacts(ctx context.Context, name, appDir string, cfg *storage.AppConfig) {
	printInfo("Removing Caddy snippet...")
	if err := caddy.RemoveSnippet(caddy.ResolveConfDir(), name); err != nil {
		printWarning(fmt.Sprintf("removing caddy snippet returned error: %v", err))
	}

	if cfg.IsCompose() {
		destroyComposeWorkload(ctx, appDir, cfg)
	} else if !cfg.IsStatic() {
		imageName := fmt.Sprintf("localhost/%s:latest", name)
		printInfo(fmt.Sprintf("Removing container image %s...", imageName))
		if err := builder.RemoveImage(ctx, imageName); err != nil {
			printWarning(fmt.Sprintf("removing container image returned error: %v", err))
		}
	}

	printInfo(fmt.Sprintf("Removing app storage at %s...", appDir))
	if err := storage.DeleteAppStorage(appDir); err != nil {
		printWarning(fmt.Sprintf("removing storage returned error: %v", err))
	}
}

func reloadDaemons(ctx context.Context) {
	printInfo("Reloading systemd daemon...")
	if err := systemd.DaemonReload(ctx); err != nil {
		printWarning(fmt.Sprintf("daemon-reload error: %v", err))
	}

	printInfo("Reloading Caddy...")
	if err := caddy.Reload(ctx); err != nil {
		printWarning(fmt.Sprintf("caddy reload error: %v", err))
	}
}
