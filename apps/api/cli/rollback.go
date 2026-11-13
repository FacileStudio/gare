package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/FacileStudio/gare/apps/api/internal/deploys"
	"github.com/FacileStudio/gare/apps/api/internal/engine"
	"github.com/spf13/cobra"
)

// NewRollbackCmd builds the rollback command.
func NewRollbackCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rollback <name> [<deploy id>]",
		Short: "Run the image of an earlier successful deploy (container workloads only)",
		Long: "Run the image an earlier successful deploy built, without rebuilding: the given deploy,\n" +
			"or the newest successful deploy of another commit than the one running now. The rollback is\n" +
			"recorded as a deploy of its own. Only container workloads keep earlier images (the last 5),\n" +
			"so static and compose apps are refused. A deploy whose health check fails rolls back by itself.",
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Minute)
			defer cancel()
			opts := engine.RollbackOptions{Trigger: deploys.TriggerCLI}
			if len(args) == 2 {
				opts.To = args[1]
			}
			result, err := newEngine(ctx).Rollback(ctx, args[0], opts)
			if err != nil {
				return err
			}
			printSuccess(fmt.Sprintf("Rolled back %s to %s -> %s", result.Name, result.Commit, result.Target))
			return nil
		},
	}
}
