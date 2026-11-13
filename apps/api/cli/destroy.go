package cli

import (
	"context"
	"time"

	"github.com/spf13/cobra"
)

// NewDestroyCmd builds the destroy command.
func NewDestroyCmd() *cobra.Command {
	opts := destroyOptions{}
	cmd := &cobra.Command{
		Use:     "destroy <name>",
		Aliases: []string{"delete", "rm"},
		Short:   "Stop and tear down an application and its resources",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 2*time.Minute)
			defer cancel()
			return runDestroy(ctx, args[0], opts)
		},
	}
	cmd.Flags().BoolVarP(&opts.yes, "yes", "y", false, "Destroy without asking for confirmation")
	cmd.Flags().BoolVar(&opts.volumes, "volumes", false, "Also delete a compose stack's named volumes and their data")
	return cmd
}
