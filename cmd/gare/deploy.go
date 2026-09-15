package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/FacileStudio/gare/internal/atomicfile"
	"github.com/FacileStudio/gare/internal/builder"
	"github.com/FacileStudio/gare/internal/caddy"
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
	if err := buildAppImage(ctx, name, repoDir); err != nil {
		return err
	}

	if err := syncRepoManifest(appDir, repoDir); err != nil {
		return err
	}

	if err := restartAppServices(ctx, name); err != nil {
		return err
	}

	cleanupAppDeploy(ctx, repoDir, name, cfg.Domain, cfg.Port)
	return nil
}

func syncRepoManifest(appDir, repoDir string) error {
	repoManifest := filepath.Join(repoDir, "manifest.yaml")
	data, err := os.ReadFile(repoManifest)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read repo manifest: %w", err)
	}

	appManifest := storage.GetManifestPath(appDir)
	if err := atomicfile.WriteFile(appManifest, data, 0644); err != nil {
		return fmt.Errorf("failed to sync manifest: %w", err)
	}
	printSuccess("Synced manifest.yaml from repository")
	return nil
}

func buildAppImage(ctx context.Context, name, repoDir string) error {
	printInfo(fmt.Sprintf("Pulling latest git changes for %s...", name))
	if err := builder.Pull(ctx, repoDir, os.Stdout, os.Stderr); err != nil {
		return fmt.Errorf("git pull failed: %w", err)
	}

	if _, err := builder.DetectContainerfile(repoDir); err != nil {
		return fmt.Errorf("containerfile detection failed: %w", err)
	}

	imageName := fmt.Sprintf("localhost/%s:latest", name)
	printInfo(fmt.Sprintf("Building container image %s...", imageName))
	if err := builder.Build(ctx, repoDir, imageName, os.Stdout, os.Stderr); err != nil {
		return fmt.Errorf("container build failed: %w", err)
	}
	return nil
}

func restartAppServices(ctx context.Context, name string) error {
	if err := systemd.DaemonReload(ctx); err != nil {
		return fmt.Errorf("systemctl daemon-reload failed: %w", err)
	}
	if err := systemd.Enable(ctx, name); err != nil {
		return fmt.Errorf("failed to enable service: %w", err)
	}
	if err := systemd.Restart(ctx, name); err != nil {
		return fmt.Errorf("failed to restart service: %w", err)
	}
	return nil
}

func cleanupAppDeploy(ctx context.Context, repoDir, name, domain string, port int) {
	if err := caddy.Reload(ctx); err != nil {
		printWarning(fmt.Sprintf("Caddy reload returned error: %v", err))
	}
	if err := builder.PruneImages(ctx, os.Stdout, os.Stderr); err != nil {
		printWarning(fmt.Sprintf("Image pruning returned error: %v", err))
	}
	commitHash, err := builder.GetCommitHash(ctx, repoDir)
	if err != nil || commitHash == "" {
		commitHash = "-"
	}
	printSuccess(fmt.Sprintf("Successfully deployed %s (%s) on port %d -> %s", name, commitHash, port, domain))
}
