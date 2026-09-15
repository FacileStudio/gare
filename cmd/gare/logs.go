package main

import (
	"os"

	"github.com/FacileStudio/gare/internal/storage"
	"github.com/FacileStudio/gare/internal/systemd"
	"github.com/spf13/cobra"
)

// NewLogsCmd builds the logs command.
func NewLogsCmd() *cobra.Command {
	var follow bool
	var lines int
	cmd := &cobra.Command{
		Use:   "logs <name>",
		Short: "Stream application logs from journalctl",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			name := args[0]
			if err := storage.ValidateAppName(name); err != nil {
				return err
			}
			opts := systemd.LogOptions{
				Follow: follow,
				Lines:  lines,
				Stdout: os.Stdout,
				Stderr: os.Stderr,
			}
			return systemd.StreamLogs(c.Context(), name, opts)
		},
	}

	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow log stream in real-time")
	cmd.Flags().IntVarP(&lines, "lines", "n", 0, "Number of journal lines to show")
	return cmd
}
