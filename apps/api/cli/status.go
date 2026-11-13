package cli

import (
	"context"
	"io"
	"time"

	"github.com/spf13/cobra"
)

type statusOptions struct {
	jsonOutput bool
}

// NewStatusCmd builds the status command.
func NewStatusCmd() *cobra.Command {
	opts := statusOptions{}
	cmd := &cobra.Command{
		Use:   "status <name>",
		Short: "Display detailed status for an application",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
			defer cancel()
			return runStatus(ctx, cmd.OutOrStdout(), args[0], opts)
		},
	}
	cmd.Flags().BoolVar(&opts.jsonOutput, "json", false, "Output JSON object")
	return cmd
}

// runStatus renders the engine's status of one application, as JSON or as a tree.
func runStatus(ctx context.Context, w io.Writer, name string, opts statusOptions) error {
	details, err := newEngine(ctx).Status(ctx, name)
	if err != nil {
		return err
	}
	if opts.jsonOutput {
		return writeJSON(w, details)
	}
	return outputStatusHuman(w, details)
}
