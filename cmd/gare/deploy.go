package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/FacileStudio/gare/internal/git"
	"github.com/FacileStudio/gare/internal/podman"
	"github.com/FacileStudio/gare/internal/storage"
	"github.com/FacileStudio/gare/internal/systemd"
	"github.com/spf13/cobra"
)

// NewDeployCmd builds the deploy command.
func NewDeployCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "deploy <name>",
		Short: "Build and deploy an application workload",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Minute)
			defer cancel()
			return RunDeploy(ctx, args[0])
		},
	}
}

// RunDeploy executes the deployment pipeline for a single application. Every workload type runs the
// same tail: prepare its artifacts, sync ingress, restart the unit, verify health, then clean up.
func RunDeploy(ctx context.Context, name string) error {
	baseDir := storage.DefaultBaseDir()
	appDir, cfg, err := loadAppConfig(name)
	if err != nil {
		return err
	}

	repoDir := storage.GetRepoDir(appDir)
	if err := fetchAppRevision(ctx, baseDir, appDir, repoDir, cfg); err != nil {
		return err
	}
	printVerbose(ctx, "Resolved %s app %s: repo %s, port %d, container port %d, unit %s",
		cfg.AppType, name, repoDir, cfg.Port, cfg.ContainerPort, systemd.GetUnitPath(name))
	if err := prepareWorkload(ctx, name, appDir, repoDir, cfg); err != nil {
		return err
	}
	if err := activateApp(ctx, cfg, repoDir); err != nil {
		return err
	}
	printSuccess(fmt.Sprintf("Successfully deployed %s app %s (%s) -> %s",
		cfg.AppType, name, deployedCommit(ctx, repoDir), formatDeployTarget(cfg)))
	return nil
}

func fetchAppRevision(ctx context.Context, baseDir, appDir, repoDir string, cfg *storage.AppConfig) error {
	printInfo(fmt.Sprintf("Pulling latest git changes for %s...", cfg.Name))
	if err := git.Pull(ctx, repoDir, settingsFrom(ctx).auth, os.Stdout, os.Stderr); err != nil {
		return fmt.Errorf("git pull failed: %w", err)
	}
	return syncDeployConfig(baseDir, repoDir, appDir, cfg)
}

// activateApp syncs ingress, restarts the unit, and only reports success once the health probes pass.
func activateApp(ctx context.Context, cfg *storage.AppConfig, repoDir string) error {
	warnComposeReachability(repoDir, cfg)
	if err := syncAppIngress(cfg); err != nil {
		printWarning(fmt.Sprintf("Could not update the Caddy snippet (%v)", err))
	} else if len(cfg.Domains) > 0 {
		printVerbose(ctx, "Wrote the Caddy snippet for %s: %s", cfg.Name, strings.Join(cfg.Domains, ", "))
	} else {
		printVerbose(ctx, "Removed the Caddy snippet for %s (no domains configured)", cfg.Name)
	}
	printVerbose(ctx, "Enabling and restarting %s.service", cfg.Name)
	if err := restartAppServices(ctx, cfg); err != nil {
		return err
	}
	if err := verifyHealth(ctx, cfg); err != nil {
		return err
	}
	cleanupAppDeploy(ctx)
	return nil
}

// prepareWorkload writes the artifacts the configured workload type needs before it restarts.
func prepareWorkload(ctx context.Context, name, appDir, repoDir string, cfg *storage.AppConfig) error {
	if cfg.IsStatic() {
		return prepareStaticDeploy(ctx, name, appDir, repoDir, cfg)
	}
	if cfg.IsCompose() {
		return prepareComposeDeploy(ctx, name, appDir, repoDir, cfg)
	}
	return prepareContainerDeploy(ctx, name, appDir, repoDir, cfg)
}

func syncDeployConfig(baseDir, repoDir, appDir string, cfg *storage.AppConfig) error {
	if err := syncGareFileConfig(baseDir, repoDir, cfg); err != nil {
		return err
	}
	warnStrayComposeFile(repoDir, cfg.IsCompose())
	if cfg.UsesPodManifest() && cfg.Port == 0 {
		port, err := storage.DiscoverAvailablePort(baseDir, 0)
		if err != nil {
			return fmt.Errorf("failed to discover free port: %w", err)
		}
		cfg.Port = port
	}
	if cfg.UsesPodManifest() && cfg.ContainerPort == 0 {
		if exposed := podman.DetectExposedPort(repoDir, cfg.Containerfile); exposed > 0 {
			cfg.ContainerPort = exposed
		}
	}
	if err := storage.SaveConfig(appDir, cfg); err != nil {
		printWarning(fmt.Sprintf("Could not persist updated config (%v)", err))
	}
	return nil
}

func deployedCommit(ctx context.Context, repoDir string) string {
	commit, _ := git.GetCommitHash(ctx, repoDir)
	if commit == "" {
		return "-"
	}
	return commit
}
