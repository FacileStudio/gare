package cli

import (
	"context"
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/FacileStudio/gare/apps/api/internal/deploys"
	"github.com/spf13/cobra"
)

// deploysOptions holds the flags of the deploys command.
type deploysOptions struct {
	jsonOutput bool
	showLog    bool
}

// NewDeploysCmd builds the deploys command, which reads an app's deploy history.
func NewDeploysCmd() *cobra.Command {
	opts := deploysOptions{}
	cmd := &cobra.Command{
		Use:   "deploys <name> [<deploy id>]",
		Short: "List an application's deploys, or show one deploy and its log",
		Long: "List the last deploys of an application, newest first. With a deploy id, show that\n" +
			"deploy; with --log, print its log, following it until the deploy finishes.",
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			history, err := newEngine(cmd.Context()).History(args[0])
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()
			if len(args) == 1 {
				return listDeploys(w, history, opts.jsonOutput)
			}
			if opts.showLog {
				return followDeployLog(cmd.Context(), w, history, args[1])
			}
			return showDeploy(w, history, args[1], opts.jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&opts.jsonOutput, "json", false, "Output JSON")
	cmd.Flags().BoolVar(&opts.showLog, "log", false, "Print the deploy's log, following it while it runs")
	return cmd
}

// listDeploys prints an app's deploy history as a table, or as a JSON array.
func listDeploys(w io.Writer, history deploys.History, jsonOutput bool) error {
	list, err := history.List()
	if err != nil {
		return fmt.Errorf("failed to read the deploy history: %w", err)
	}
	if jsonOutput {
		return writeJSON(w, list)
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tSTATUS\tTRIGGER\tCOMMIT\tSTARTED\tDURATION")
	for _, d := range list {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n", d.ID, d.Status, d.Trigger, orDash(d.Commit),
			d.StartedAt.Local().Format(time.DateTime), deployDuration(d))
	}
	return tw.Flush()
}

// showDeploy prints one deploy's record.
func showDeploy(w io.Writer, history deploys.History, id string, jsonOutput bool) error {
	d, err := history.Get(id)
	if err != nil {
		return err
	}
	if jsonOutput {
		return writeJSON(w, d)
	}
	fmt.Fprintf(w, "%s %s (%s) commit %s, %s\n", d.ID, d.Status, d.Trigger, orDash(d.Commit), deployDuration(d))
	if d.RolledBackTo != nil {
		fmt.Fprintf(w, "rolled back to %s\n", *d.RolledBackTo)
	}
	if d.Error != nil {
		fmt.Fprintf(w, "error: %s\n", *d.Error)
	}
	return nil
}

// followDeployLog prints a deploy's log and keeps printing while it runs, then its final status.
func followDeployLog(ctx context.Context, w io.Writer, history deploys.History, id string) error {
	d, err := history.Follow(ctx, id, 0, func(_ int, text string) error {
		_, err := fmt.Fprintln(w, text)
		return err
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "deploy %s %s\n", d.ID, d.Status)
	return nil
}

// deployDuration is how long a deploy ran, or its status while it is queued or running.
func deployDuration(d deploys.Deploy) string {
	if d.FinishedAt == nil {
		return string(d.Status)
	}
	return d.FinishedAt.Sub(d.StartedAt).Round(time.Second).String()
}

// orDash shows a missing value as a dash.
func orDash(value string) string {
	if value == "" {
		return "-"
	}
	return value
}
