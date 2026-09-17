package main

import (
	"context"
	"fmt"
	"time"

	"github.com/FacileStudio/gare/internal/caddy"
	"github.com/FacileStudio/gare/internal/storage"
	"github.com/FacileStudio/gare/internal/systemd"
	"github.com/spf13/cobra"
)

// NewStartCmd builds the start command.
func NewStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start <name>",
		Short: "Start an application workload",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			return runStartApp(ctx, args[0])
		},
	}
}

// NewStopCmd builds the stop command.
func NewStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop <name>",
		Short: "Stop an application workload",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			return runStopApp(ctx, args[0])
		},
	}
}

// NewRestartCmd builds the restart command.
func NewRestartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restart <name>",
		Short: "Restart an application workload",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			return runRestartApp(ctx, args[0])
		},
	}
}

func runStartApp(ctx context.Context, name string) error {
	cfg, err := loadAppForLifecycle(name)
	if err != nil {
		return err
	}
	if cfg.IsStatic() {
		printInfo(fmt.Sprintf("App %q is static and served by Caddy", name))
		return nil
	}
	if err := systemd.Start(ctx, name); err != nil {
		return fmt.Errorf("failed to start %q: %w", name, err)
	}
	printSuccess(fmt.Sprintf("Started %s.service", name))
	return nil
}

func runStopApp(ctx context.Context, name string) error {
	cfg, err := loadAppForLifecycle(name)
	if err != nil {
		return err
	}
	if cfg.IsStatic() {
		printInfo(fmt.Sprintf("App %q is static and served by Caddy", name))
		return nil
	}
	if err := systemd.Stop(ctx, name); err != nil {
		return fmt.Errorf("failed to stop %q: %w", name, err)
	}
	printSuccess(fmt.Sprintf("Stopped %s.service", name))
	return nil
}

func runRestartApp(ctx context.Context, name string) error {
	cfg, err := loadAppForLifecycle(name)
	if err != nil {
		return err
	}
	if cfg.IsStatic() {
		if err := caddy.Reload(ctx); err != nil {
			return fmt.Errorf("failed to reload caddy: %w", err)
		}
		printSuccess("Reloaded Caddy configuration for static site")
		return nil
	}
	if err := systemd.Restart(ctx, name); err != nil {
		return fmt.Errorf("failed to restart %q: %w", name, err)
	}
	printSuccess(fmt.Sprintf("Restarted %s.service", name))
	return nil
}

func loadAppForLifecycle(name string) (*storage.AppConfig, error) {
	if err := storage.ValidateAppName(name); err != nil {
		return nil, err
	}
	baseDir := storage.DefaultBaseDir()
	appDir := storage.GetAppDir(baseDir, name)
	cfg, err := storage.LoadConfig(appDir)
	if err != nil {
		return nil, fmt.Errorf("app %q not found: %w", name, err)
	}
	return cfg, nil
}
