package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/FacileStudio/gare/internal/builder"
	"github.com/FacileStudio/gare/internal/storage"
	"github.com/FacileStudio/gare/internal/systemd"
	"github.com/spf13/cobra"
)

type listOptions struct {
	jsonOutput  bool
	quietOutput bool
}

type appListItem struct {
	Name      string `json:"name"`
	RepoURL   string `json:"repo_url"`
	Domain    string `json:"domain"`
	Port      int    `json:"port"`
	Branch    string `json:"branch"`
	CreatedAt string `json:"created_at"`
	Commit    string `json:"commit"`
	Status    string `json:"status"`
}

// NewListCmd builds the list command.
func NewListCmd() *cobra.Command {
	opts := listOptions{}
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all configured applications",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			return runList(ctx, cmd.OutOrStdout(), opts)
		},
	}

	cmd.Flags().BoolVar(&opts.jsonOutput, "json", false, "Output JSON array")
	cmd.Flags().BoolVarP(&opts.quietOutput, "quiet", "q", false, "Output bare app names")
	return cmd
}

func runList(ctx context.Context, w io.Writer, opts listOptions) error {
	baseDir := storage.DefaultBaseDir()
	apps, err := storage.ListApps(baseDir)
	if err != nil {
		return fmt.Errorf("failed to list apps: %w", err)
	}

	if opts.quietOutput {
		return outputQuiet(w, apps)
	}

	items := collectAppItems(ctx, baseDir, apps)
	if opts.jsonOutput {
		return outputJSON(w, items)
	}
	return outputTable(w, items)
}

func collectAppItems(ctx context.Context, baseDir string, apps []*storage.AppConfig) []appListItem {
	items := make([]appListItem, 0, len(apps))
	for _, app := range apps {
		appDir := storage.GetAppDir(baseDir, app.Name)
		repoDir := storage.GetRepoDir(appDir)
		commit, _ := builder.GetCommitHash(ctx, repoDir)
		status, _ := systemd.IsActive(ctx, app.Name)
		if status == "" {
			if app.IsStatic() {
				status = "static"
			} else {
				status = "inactive"
			}
		}
		items = append(items, appListItem{
			Name:      app.Name,
			RepoURL:   app.RepoURL,
			Domain:    app.Domain,
			Port:      app.Port,
			Branch:    app.Branch,
			CreatedAt: app.CreatedAt,
			Commit:    commit,
			Status:    status,
		})
	}
	return items
}

func outputQuiet(w io.Writer, apps []*storage.AppConfig) error {
	for _, app := range apps {
		fmt.Fprintln(w, app.Name)
	}
	return nil
}

func outputJSON(w io.Writer, items []appListItem) error {
	if items == nil {
		items = []appListItem{}
	}
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(w, string(data))
	return nil
}

func outputTable(w io.Writer, items []appListItem) error {
	if len(items) == 0 {
		fmt.Fprintln(w, "No applications configured")
		return nil
	}

	rows := make([]AppRow, len(items))
	for i, item := range items {
		commit := item.Commit
		if commit == "" {
			commit = "-"
		}
		rows[i] = AppRow{
			Name:   item.Name,
			Port:   item.Port,
			Domain: item.Domain,
			Commit: commit,
			Status: item.Status,
		}
	}
	fmt.Fprintln(w, renderAppTable(rows))
	return nil
}
