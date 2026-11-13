package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/FacileStudio/gare/apps/api/internal/deploys"
	"github.com/FacileStudio/gare/apps/api/internal/engine"
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
			return RunDeploy(ctx, args[0], deploys.TriggerCLI)
		},
	}
}

// RunDeploy deploys one application through the engine and prints the line that reports it. The
// webhook daemon calls it too, so a pushed deploy prints exactly what a typed one does; trigger
// records which of them started it.
func RunDeploy(ctx context.Context, name string, trigger deploys.Trigger) error {
	result, err := newEngine(ctx).Deploy(ctx, name, engine.DeployOptions{Trigger: trigger})
	if err != nil {
		return err
	}
	printSuccess(fmt.Sprintf("Successfully deployed %s app %s (%s) -> %s",
		result.AppType, result.Name, result.Commit, result.Target))
	return nil
}
