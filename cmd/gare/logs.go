package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/FacileStudio/gare/internal/storage"
	"github.com/FacileStudio/gare/internal/systemd"
	"github.com/spf13/cobra"
)

const defaultLogLines = 100

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
	cmd.Flags().IntVarP(&lines, "lines", "n", defaultLogLines, "Number of journal lines to show")
	return cmd
}

func runLogs(ctx context.Context, name string, follow bool, lines int) error {
	if err := storage.ValidateAppName(name); err != nil {
		return err
	}
	baseDir := storage.DefaultBaseDir()
	appDir := storage.GetAppDir(baseDir, name)
	if _, err := storage.LoadConfig(appDir); err != nil {
		return appConfigError(name, err)
	}
	unitPath := systemd.GetUnitPath(name)
	if _, err := os.Stat(unitPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("app %q has not been deployed yet", name)
		}
		return err
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
