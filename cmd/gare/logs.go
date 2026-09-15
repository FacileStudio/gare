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
	cmd := &cobra.Command{
		Use:   "logs <name>",
		Short: "Stream application logs from journalctl",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			name := args[0]
			if err := storage.ValidateAppName(name); err != nil {
				return err
			}
			return systemd.StreamLogs(c.Context(), name, follow, os.Stdout, os.Stderr)
		},
	}

	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow log stream in real-time")
	return cmd
}
