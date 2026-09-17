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
	if err := storage.SaveConfig(appDir, cfg); err != nil {
		printWarning(fmt.Sprintf("Could not persist updated config (%v)", err))
	}
	return nil
}

func deployStaticApp(ctx context.Context, name, repoDir string, cfg *storage.AppConfig) error {
	if cfg.BuildCmd != "" {
		printInfo(fmt.Sprintf("Running build command: %s", cfg.BuildCmd))
		if err := builder.RunBuildCommand(ctx, repoDir, cfg.BuildCmd, os.Stdout, os.Stderr); err != nil {
			return fmt.Errorf("build command failed: %w", err)
		}
	}

	staticPath := filepath.Join(repoDir, cfg.StaticDir)
	if cfg.Domain != "" || cfg.Port > 0 {
		if err := caddy.WriteStaticSnippet(caddy.ResolveConfDir(), name, cfg.Domain, cfg.Port, staticPath); err != nil {
			printWarning(fmt.Sprintf("Could not write Caddy snippet (%v)", err))
		}
	}

	cleanupAppDeploy(ctx)

	commitHash, _ := builder.GetCommitHash(ctx, repoDir)
	if commitHash == "" {
		commitHash = "-"
	}
	target := "static"
	if cfg.Domain != "" {
		target = cfg.Domain
	} else if cfg.Port > 0 {
		target = fmt.Sprintf("port %d", cfg.Port)
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

func updateContainerIngress(name string, cfg *storage.AppConfig) {
	if cfg.Domain != "" && cfg.Port > 0 {
		if err := caddy.WriteSnippet(caddy.ResolveConfDir(), name, cfg.Domain, cfg.Port); err != nil {
			printWarning(fmt.Sprintf("Could not update Caddy snippet (%v)", err))
		}
	}
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
