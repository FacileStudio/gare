package cli

import (
	"context"
	"fmt"
	"time"

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

// runStartApp starts an application and reports it active.
func runStartApp(ctx context.Context, name string) error {
	if err := newEngine(ctx).Start(ctx, name); err != nil {
		return err
	}
	printSuccess(fmt.Sprintf("Started %s.service (active)", name))
	return nil
}

// runStopApp stops an application and reports the state systemd settled in.
func runStopApp(ctx context.Context, name string) error {
	result, err := newEngine(ctx).Stop(ctx, name)
	if err != nil {
		return err
	}
	printSuccess(fmt.Sprintf("Stopped %s.service (%s)", name, result.ActiveState))
	return nil
}

// runRestartApp restarts an application and reports it active.
func runRestartApp(ctx context.Context, name string) error {
	if err := newEngine(ctx).Restart(ctx, name); err != nil {
		return err
	}
	printSuccess(fmt.Sprintf("Restarted %s.service (active)", name))
	return nil
}
