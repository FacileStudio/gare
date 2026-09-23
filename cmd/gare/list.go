package main

import (
	"context"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/FacileStudio/gare/internal/git"
	"github.com/FacileStudio/gare/internal/storage"
	"github.com/FacileStudio/gare/internal/systemd"
	"github.com/spf13/cobra"
)

type listOptions struct {
	jsonOutput  bool
	quietOutput bool
	tag         string
}

type appListItem struct {
	Name      string   `json:"name"`
	RepoURL   string   `json:"repo_url"`
	Domains   []string `json:"domains,omitempty"`
	Port      int      `json:"port,omitempty"`
	Tags      []string `json:"tags,omitempty"`
	Branch    string   `json:"branch"`
	CreatedAt string   `json:"created_at"`
	Commit    string   `json:"commit"`
	Status    string   `json:"status"`
}

// NewListCmd builds the list command.
func NewListCmd() *cobra.Command {
	opts := listOptions{}
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls", "ps"},
		Short:   "List all managed applications",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
			defer cancel()
			return runList(ctx, cmd.OutOrStdout(), opts)
		},
	}

	cmd.Flags().BoolVar(&opts.jsonOutput, "json", false, "Output JSON array")
	cmd.Flags().BoolVarP(&opts.quietOutput, "quiet", "q", false, "Output application names only")
	cmd.Flags().StringVarP(&opts.tag, "tag", "t", "", "Filter applications by tag")
	return cmd
}

func runList(ctx context.Context, w io.Writer, opts listOptions) error {
	baseDir := storage.DefaultBaseDir()
	apps, err := storage.ListApps(baseDir)
	if err != nil {
		return fmt.Errorf("failed to list apps: %w", err)
	}

	if opts.tag != "" {
		apps = storage.FilterAppsByTag(apps, opts.tag)
	}

	if opts.quietOutput {
		return outputQuiet(w, apps)
	}

	items := collectAppItems(ctx, baseDir, apps)
	if opts.jsonOutput {
		return writeJSON(w, items)
	}
	return outputTable(w, items)
}

func collectAppItems(ctx context.Context, baseDir string, apps []*storage.AppConfig) []appListItem {
	items := make([]appListItem, 0, len(apps))
	for _, app := range apps {
		appDir := storage.GetAppDir(baseDir, app.Name)
		repoDir := storage.GetRepoDir(appDir)
		commit, _ := git.GetCommitHash(ctx, repoDir)
		status := "inactive"
		s, _ := systemd.IsActive(ctx, app.Name)
		if s != "" {
			status = s
		}
		items = append(items, appListItem{
			Name:      app.Name,
			RepoURL:   app.RepoURL,
			Domains:   app.Domains,
			Port:      app.Port,
			Tags:      app.Tags,
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

func outputTable(w io.Writer, items []appListItem) error {
	if len(items) == 0 {
		fmt.Fprintln(w, "No applications configured")
		return nil
	}

	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
	fmt.Fprintln(tw, "NAME\tSTATUS\tDOMAIN\tPORT\tTAGS\tBRANCH\tCOMMIT\tCREATED")
	for _, item := range items {
		domain := "-"
		if len(item.Domains) > 0 {
			domain = strings.Join(item.Domains, ", ")
		}
		portStr := fmt.Sprintf("%d", item.Port)
		if item.Port == 0 {
			portStr = "-"
		}
		tagsStr := "-"
		if len(item.Tags) > 0 {
			tagsStr = strings.Join(item.Tags, ",")
		}
		created := renderCreatedAt(item.CreatedAt)
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			item.Name, item.Status, domain, portStr, tagsStr, item.Branch, item.Commit, created)
	}
	return tw.Flush()
}

// renderCreatedAt shows the stored UTC timestamp in the operator's own timezone, so an app created
// seconds ago does not read as hours old. A value the storage layer cannot parse is shown as stored.
func renderCreatedAt(created string) string {
	parsed, err := time.Parse(time.RFC3339, created)
	if err != nil {
		return created
	}
	return parsed.Local().Format("2006-01-02 15:04")
}
