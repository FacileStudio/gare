package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/FacileStudio/gare/apps/api/internal/git"
	"github.com/FacileStudio/gare/apps/api/internal/podman"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
	"github.com/FacileStudio/gare/apps/api/internal/systemd"
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
	store := storage.Default()
	appDir, cfg, err := loadAppConfig(name)
	if err != nil {
		return err
	}

	repoDir := storage.GetRepoDir(appDir)
	if err := fetchAppRevision(ctx, store, repoDir, cfg); err != nil {
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

func fetchAppRevision(ctx context.Context, store *storage.Store, repoDir string, cfg *storage.AppConfig) error {
	printInfo(fmt.Sprintf("Syncing %s with its git remote...", cfg.Name))
	opts := git.SyncOptions{RepoDir: repoDir, Branch: cfg.Branch, Auth: settingsFrom(ctx).auth,
		Stdout: os.Stdout, Stderr: os.Stderr}
	if err := git.Sync(ctx, opts); err != nil {
		return fmt.Errorf("git sync failed: %w", err)
	}
	return syncDeployConfig(store, repoDir, cfg)
}

// activateApp syncs ingress, restarts the unit, and only reports success once the ingress serves the
// app's domains and the health probes pass.
func activateApp(ctx context.Context, cfg *storage.AppConfig, repoDir string) error {
	warnComposeReachability(repoDir, cfg)
	if err := activateIngress(ctx, cfg); err != nil {
		return err
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

func syncDeployConfig(store *storage.Store, repoDir string, cfg *storage.AppConfig) error {
	if err := resolveDeployConfig(store, repoDir, cfg); err != nil {
		return err
	}
	warnStrayComposeFile(repoDir, cfg.IsCompose())
	if cfg.UsesPodManifest() && cfg.Port == 0 {
		port, err := store.AvailablePort(0)
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
	if err := store.Save(cfg); err != nil {
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
