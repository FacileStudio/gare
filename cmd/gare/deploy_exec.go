package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/FacileStudio/gare/internal/caddy"
	"github.com/FacileStudio/gare/internal/podman"
	"github.com/FacileStudio/gare/internal/storage"
	"github.com/FacileStudio/gare/internal/systemd"
)

// prepareStaticDeploy runs the static workload's build step and writes the unit serving its directory.
func prepareStaticDeploy(ctx context.Context, name, appDir, repoDir string, cfg *storage.AppConfig) error {
	if err := runBuildCommand(ctx, repoDir, cfg.BuildCmd); err != nil {
		return err
	}
	return writeStaticUnit(ctx, name, appDir, filepath.Join(repoDir, cfg.StaticDir), cfg.Port)
}

// prepareContainerDeploy builds the application image, syncs its manifest and writes its unit.
func prepareContainerDeploy(ctx context.Context, name, appDir, repoDir string, cfg *storage.AppConfig) error {
	if err := runBuildCommand(ctx, repoDir, cfg.BuildCmd); err != nil {
		return err
	}
	if err := buildAppImage(ctx, name, repoDir, cfg); err != nil {
		return err
	}
	if err := syncManifest(name, appDir, repoDir, cfg); err != nil {
		return err
	}
	return writeContainerUnit(ctx, name, appDir)
}

// runBuildCommand runs the repository's configured build command, if it has one.
func runBuildCommand(ctx context.Context, repoDir, command string) error {
	if command == "" {
		return nil
	}
	printInfo(fmt.Sprintf("Running build command: %s", command))
	if err := podman.RunBuildCommand(ctx, repoDir, command, os.Stdout, os.Stderr); err != nil {
		return fmt.Errorf("build command failed: %w", err)
	}
	return nil
}

func formatDeployTarget(cfg *storage.AppConfig) string {
	if len(cfg.Domains) > 0 {
		return fmt.Sprintf("%s (:%d)", strings.Join(cfg.Domains, ", "), cfg.Port)
	}
	return fmt.Sprintf("port %d", cfg.Port)
}

func buildAppImage(ctx context.Context, name, repoDir string, cfg *storage.AppConfig) error {
	imageName := fmt.Sprintf("localhost/%s:latest", name)
	printInfo(fmt.Sprintf("Building container image %s...", imageName))
	buildOpts := podman.BuildOptions{
		RepoDir:       repoDir,
		ImageName:     imageName,
		Containerfile: cfg.Containerfile,
		ContextDir:    cfg.ContextDir,
		Stdout:        os.Stdout,
		Stderr:        os.Stderr,
	}
	if err := podman.Build(ctx, buildOpts); err != nil {
		return fmt.Errorf("container build failed: %w", err)
	}
	return nil
}

func restartAppServices(ctx context.Context, cfg *storage.AppConfig) error {
	if err := systemd.DaemonReload(ctx); err != nil {
		return fmt.Errorf("systemctl daemon-reload failed: %w", err)
	}
	if err := systemd.Enable(ctx, cfg.Name); err != nil {
		return fmt.Errorf("failed to enable service: %w", err)
	}
	if err := systemd.Restart(ctx, cfg.Name); err != nil {
		return fmt.Errorf("failed to restart service: %w", err)
	}
	return nil
}

func cleanupAppDeploy(ctx context.Context) {
	if err := caddy.Reload(ctx); err != nil {
		printWarning(fmt.Sprintf("Caddy reload returned error: %v", err))
	}
	if err := podman.PruneImages(ctx, os.Stdout, os.Stderr); err != nil {
		printWarning(fmt.Sprintf("Image pruning returned error: %v", err))
	}
}
