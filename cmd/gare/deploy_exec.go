package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/FacileStudio/gare/internal/builder"
	"github.com/FacileStudio/gare/internal/caddy"
	"github.com/FacileStudio/gare/internal/health"
	"github.com/FacileStudio/gare/internal/storage"
	"github.com/FacileStudio/gare/internal/systemd"
)

func prepareStaticDeploy(ctx context.Context, name, repoDir string, cfg *storage.AppConfig) error {
	if cfg.BuildCmd != "" {
		printInfo(fmt.Sprintf("Running build command: %s", cfg.BuildCmd))
		if err := builder.RunBuildCommand(ctx, repoDir, cfg.BuildCmd, os.Stdout, os.Stderr); err != nil {
			return fmt.Errorf("build command failed: %w", err)
		}
	}
	staticPath := filepath.Join(repoDir, cfg.StaticDir)
	if err := systemd.WriteStaticUnit(name, cfg.Port, staticPath); err != nil {
		return fmt.Errorf("failed to write systemd unit: %w", err)
	}
	if cfg.Domain != "" && cfg.Port > 0 {
		if err := caddy.WriteSnippet(caddy.ResolveConfDir(), name, cfg.Domain, cfg.Port); err != nil {
			printWarning(fmt.Sprintf("Could not write Caddy snippet (%v)", err))
		}
	}
	return nil
}

func formatDeployTarget(cfg *storage.AppConfig) string {
	if cfg.Domain != "" {
		return fmt.Sprintf("%s (:%d)", cfg.Domain, cfg.Port)
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
	return syncRepoManifest(appDir, repoDir)
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

func updateContainerIngress(name string, cfg *storage.AppConfig) {
	if cfg.Domain != "" && cfg.Port > 0 {
		if err := caddy.WriteSnippet(caddy.ResolveConfDir(), name, cfg.Domain, cfg.Port); err != nil {
			printWarning(fmt.Sprintf("Could not update Caddy snippet (%v)", err))
		}
	}
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

func verifyHealth(ctx context.Context, cfg *storage.AppConfig) error {
	if cfg.Port <= 0 {
		return nil
	}
	time.Sleep(100 * time.Millisecond)
	props, _ := systemd.GetServiceProperties(ctx, cfg.Name)
	if props != nil && props.ActiveState != "active" {
		return fmt.Errorf("service %s failed to start (state: %s)", cfg.Name, props.ActiveState)
	}
	healthPath := cfg.Healthcheck
	if healthPath == "" {
		healthPath = "/"
	}
	if !strings.HasPrefix(healthPath, "/") {
		healthPath = "/" + healthPath
	}
	probeURL := fmt.Sprintf("http://127.0.0.1:%d%s", cfg.Port, healthPath)
	printInfo(fmt.Sprintf("Verifying health probe at %s...", probeURL))
	if err := health.Probe(ctx, probeURL, 30*time.Second); err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	printSuccess("Health check passed")
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
