package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/FacileStudio/gare/internal/builder"
	"github.com/FacileStudio/gare/internal/caddy"
	"github.com/FacileStudio/gare/internal/storage"
	"github.com/FacileStudio/gare/internal/systemd"
)

func prepareStaticDeploy(ctx context.Context, name, appDir, repoDir string, cfg *storage.AppConfig) error {
	if cfg.BuildCmd != "" {
		printInfo(fmt.Sprintf("Running build command: %s", cfg.BuildCmd))
		if err := builder.RunBuildCommand(ctx, repoDir, cfg.BuildCmd, os.Stdout, os.Stderr); err != nil {
			return fmt.Errorf("build command failed: %w", err)
		}
	}
	staticPath := filepath.Join(repoDir, cfg.StaticDir)
	if err := writeStaticUnit(name, appDir, staticPath, cfg.Port); err != nil {
		return err
	}
	if err := syncAppIngress(cfg); err != nil {
		printWarning(fmt.Sprintf("Could not write Caddy snippet (%v)", err))
	}
	return nil
}

func formatDeployTarget(cfg *storage.AppConfig) string {
	if len(cfg.Domains) > 0 {
		return fmt.Sprintf("%s (:%d)", strings.Join(cfg.Domains, ", "), cfg.Port)
	}
	return fmt.Sprintf("port %d", cfg.Port)
}

func executePreDeploy(ctx context.Context, name, appDir, repoDir string, cfg *storage.AppConfig) error {
	if cfg.BuildCmd != "" {
		printInfo(fmt.Sprintf("Running build command: %s", cfg.BuildCmd))
		if err := builder.RunBuildCommand(ctx, repoDir, cfg.BuildCmd, os.Stdout, os.Stderr); err != nil {
			return fmt.Errorf("build command failed: %w", err)
		}
	}
	if err := buildAppImage(ctx, name, repoDir, cfg); err != nil {
		return err
	}
	if err := syncManifest(name, appDir, repoDir, cfg); err != nil {
		return err
	}
	return writeContainerUnit(name, appDir)
}

func buildAppImage(ctx context.Context, name, repoDir string, cfg *storage.AppConfig) error {
	imageName := fmt.Sprintf("localhost/%s:latest", name)
	printInfo(fmt.Sprintf("Building container image %s...", imageName))
	buildOpts := builder.BuildOptions{
		RepoDir:       repoDir,
		ImageName:     imageName,
		Containerfile: cfg.Containerfile,
		ContextDir:    cfg.ContextDir,
		Stdout:        os.Stdout,
		Stderr:        os.Stderr,
	}
	if err := builder.Build(ctx, buildOpts); err != nil {
		return fmt.Errorf("container build failed: %w", err)
	}
	return nil
}

func updateContainerIngress(cfg *storage.AppConfig) {
	if err := syncAppIngress(cfg); err != nil {
		printWarning(fmt.Sprintf("Could not update Caddy snippet (%v)", err))
	}
}

func restartAppServices(ctx context.Context, cfg *storage.AppConfig) error {
	if err := systemd.DaemonReload(ctx); err != nil {
		return fmt.Errorf("systemctl daemon-reload failed: %w", err)
	}
	if !cfg.UsesQuadletUnit() {
		if err := systemd.Enable(ctx, cfg.Name); err != nil {
			return fmt.Errorf("failed to enable service: %w", err)
		}
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
	if err := builder.PruneImages(ctx, os.Stdout, os.Stderr); err != nil {
		printWarning(fmt.Sprintf("Image pruning returned error: %v", err))
	}
}
