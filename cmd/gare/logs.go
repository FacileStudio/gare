package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

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
			return runLogs(c.Context(), args[0], follow, lines)
		},
	}

	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow log stream in real-time")
	cmd.Flags().IntVarP(&lines, "lines", "n", systemd.DefaultLogLines, "Number of journal lines to show")
	return cmd
}

func runLogs(ctx context.Context, name string, follow bool, lines int) error {
	if _, _, err := loadAppConfig(name); err != nil {
		return err
	}
	deployed, err := appUnitExists(name)
	if err != nil {
		return err
	}
	if !deployed {
		return fmt.Errorf("app %q has not been deployed yet", name)
	}
	sigCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	opts := systemd.LogOptions{
		Follow: follow,
		Lines:  lines,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	return systemd.StreamLogs(sigCtx, name, opts)
}
