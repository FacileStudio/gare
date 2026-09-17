package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/FacileStudio/gare/internal/builder"
	"github.com/FacileStudio/gare/internal/storage"
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

// RunDeploy executes the deployment pipeline for a single application.
func RunDeploy(ctx context.Context, name string) error {
	if err := storage.ValidateAppName(name); err != nil {
		return err
	}

	baseDir := storage.DefaultBaseDir()
	appDir := storage.GetAppDir(baseDir, name)
	cfg, err := storage.LoadConfig(appDir)
	if err != nil {
		return fmt.Errorf("app %q not found or invalid config: %w", name, err)
	}

	repoDir := storage.GetRepoDir(appDir)
	printInfo(fmt.Sprintf("Pulling latest git changes for %s...", name))
	if err := builder.Pull(ctx, repoDir, os.Stdout, os.Stderr); err != nil {
		return fmt.Errorf("git pull failed: %w", err)
	}
	if err := syncDeployConfig(baseDir, repoDir, appDir, cfg); err != nil {
		return err
	}

	if cfg.IsStatic() {
		return deployStaticApp(ctx, name, repoDir, cfg)
	}
	return deployContainerApp(ctx, name, appDir, repoDir, cfg)
}

func syncDeployConfig(baseDir, repoDir, appDir string, cfg *storage.AppConfig) error {
	if err := syncGareFileConfig(baseDir, repoDir, cfg); err != nil {
		return err
	}
	if !cfg.IsStatic() && cfg.Port == 0 {
		port, err := storage.DiscoverAvailablePort(baseDir, 0)
		if err != nil {
			return fmt.Errorf("failed to discover free port: %w", err)
		}
		cfg.Port = port
	}
	if !cfg.IsStatic() && cfg.ContainerPort == 0 {
		if exposed := builder.DetectExposedPort(repoDir, cfg.Containerfile); exposed > 0 {
			cfg.ContainerPort = exposed
		}
	}
	if err := storage.SaveConfig(appDir, cfg); err != nil {
		printWarning(fmt.Sprintf("Could not persist updated config (%v)", err))
	}
	return nil
}

func deployStaticApp(ctx context.Context, name, repoDir string, cfg *storage.AppConfig) error {
	if err := prepareStaticDeploy(ctx, name, repoDir, cfg); err != nil {
		return err
	}
	if err := restartAppServices(ctx, name); err != nil {
		return err
	}
	if err := verifyHealth(ctx, cfg); err != nil {
		return err
	}
	cleanupAppDeploy(ctx)
	target := formatDeployTarget(cfg)
	commitHash, _ := builder.GetCommitHash(ctx, repoDir)
	if commitHash == "" {
		commitHash = "-"
	}
	printSuccess(fmt.Sprintf("Successfully deployed static app %s (%s) -> %s", name, commitHash, target))
	return nil
}

func deployContainerApp(ctx context.Context, name, appDir, repoDir string, cfg *storage.AppConfig) error {
	if err := executePreDeploy(ctx, name, appDir, repoDir, cfg); err != nil {
		return err
	}

	updateContainerIngress(name, cfg)

	if err := restartAppServices(ctx, name); err != nil {
		return err
	}

	if err := verifyHealth(ctx, cfg); err != nil {
		return err
	}

	cleanupAppDeploy(ctx)
	return nil
}
