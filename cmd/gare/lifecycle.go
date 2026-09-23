package main

import (
	"context"
	"fmt"
	"time"

	"github.com/FacileStudio/gare/internal/caddy"
	"github.com/FacileStudio/gare/internal/systemd"
	"github.com/spf13/cobra"
)

const (
	startBudget   = 320 * time.Second
	stopBudget    = 90 * time.Second
	restartBudget = 400 * time.Second
)

// NewStartCmd builds the start command.
func NewStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start <name>",
		Short: "Start an application workload",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), startBudget)
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
			ctx, cancel := context.WithTimeout(cmd.Context(), stopBudget)
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
			ctx, cancel := context.WithTimeout(cmd.Context(), restartBudget)
			defer cancel()
			return runRestartApp(ctx, args[0])
		},
	}
}

func runStartApp(ctx context.Context, name string) error {
	_, cfg, err := loadAppConfig(name)
	if err != nil {
		return err
	}
	if err := syncAppIngress(cfg); err != nil {
		printWarning(fmt.Sprintf("Could not sync ingress (%v)", err))
	} else if reloadErr := caddy.Reload(ctx); reloadErr != nil {
		printWarning(fmt.Sprintf("Caddy reload error: %v", reloadErr))
	}
	if err := systemd.Start(ctx, name); err != nil {
		return fmt.Errorf("failed to start %q: %w", name, err)
	}
	if err := systemd.WaitForState(ctx, name, "active"); err != nil {
		return fmt.Errorf("failed to verify %q active state: %w", name, err)
	}
	printSuccess(fmt.Sprintf("Started %s.service (active)", name))
	return nil
}

func runStopApp(ctx context.Context, name string) error {
	_, cfg, err := loadAppConfig(name)
	if err != nil {
		return err
	}
	if err := caddy.RemoveSnippet(caddy.ResolveConfDir(), name); err == nil {
		if reloadErr := caddy.Reload(ctx); reloadErr != nil {
			printWarning(fmt.Sprintf("Caddy reload error: %v", reloadErr))
		}
	}
	if err := systemd.Stop(ctx, name); err != nil {
		return fmt.Errorf("failed to stop %q: %w", name, err)
	}
	props, err := systemd.WaitForStop(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to verify %q stopped: %w", name, err)
	}
	if props.Result != "" && props.Result != "success" {
		printWarning(fmt.Sprintf("Service %s exited with result %s (%s); inspect with: gare logs %s",
			name, props.Result, props.SubState, name))
	}
	verifyComposeContainersGone(ctx, cfg, "stop")
	printSuccess(fmt.Sprintf("Stopped %s.service (%s)", name, props.ActiveState))
	return nil
}

func runRestartApp(ctx context.Context, name string) error {
	_, cfg, err := loadAppConfig(name)
	if err != nil {
		return err
	}
	if err := syncAppIngress(cfg); err != nil {
		printWarning(fmt.Sprintf("Could not sync ingress (%v)", err))
	} else if reloadErr := caddy.Reload(ctx); reloadErr != nil {
		printWarning(fmt.Sprintf("Caddy reload error: %v", reloadErr))
	}
	if err := systemd.Restart(ctx, name); err != nil {
		return fmt.Errorf("failed to restart %q: %w", name, err)
	}
	if err := systemd.WaitForState(ctx, name, "active"); err != nil {
		return fmt.Errorf("failed to verify %q active state: %w", name, err)
	}
	printSuccess(fmt.Sprintf("Restarted %s.service (active)", name))
	return nil
}
