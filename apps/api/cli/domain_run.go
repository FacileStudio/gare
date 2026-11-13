package cli

import (
	"context"
	"fmt"
	"io"
	"sort"
	"text/tabwriter"

	"github.com/FacileStudio/gare/apps/api/internal/storage"
)

type domainItem struct {
	Hostname string `json:"hostname"`
	App      string `json:"app,omitempty"`
}

// runDomainAdd routes a hostname to an application through the engine.
func runDomainAdd(ctx context.Context, appName, hostname string) error {
	return newEngine(ctx).AddDomain(ctx, appName, hostname)
}

// runDomainRemove stops routing a hostname to an application through the engine.
func runDomainRemove(ctx context.Context, appName, hostname string) error {
	return newEngine(ctx).RemoveDomain(ctx, appName, hostname)
}

func runDomainList(ctx context.Context, w io.Writer, targetApp string, opts domainListOptions) error {
	var items []domainItem
	if targetApp != "" {
		cfg, err := newEngine(ctx).Config(targetApp)
		if err != nil {
			return err
		}
		items = collectAppDomains(cfg)
	} else {
		apps, err := storage.Default().List()
		if err != nil {
			return fmt.Errorf("failed to list apps: %w", err)
		}
		items = collectAllDomains(apps)
	}
	if opts.jsonOutput {
		return writeJSON(w, items)
	}
	return outputDomainTable(w, targetApp, items)
}

func collectAppDomains(cfg *storage.AppConfig) []domainItem {
	items := make([]domainItem, 0, len(cfg.Domains))
	for _, d := range cfg.Domains {
		items = append(items, domainItem{Hostname: d})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Hostname < items[j].Hostname
	})
	return items
}

func collectAllDomains(apps []*storage.AppConfig) []domainItem {
	items := make([]domainItem, 0)
	for _, app := range apps {
		for _, d := range app.Domains {
			items = append(items, domainItem{Hostname: d, App: app.Name})
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Hostname != items[j].Hostname {
			return items[i].Hostname < items[j].Hostname
		}
		return items[i].App < items[j].App
	})
	return items
}

func outputDomainTable(w io.Writer, targetApp string, items []domainItem) error {
	if len(items) == 0 {
		if targetApp != "" {
			fmt.Fprintf(w, "No domains configured for app %q\n", targetApp)
		} else {
			fmt.Fprintln(w, "No domains configured")
		}
		return nil
	}
	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
	if targetApp != "" {
		fmt.Fprintln(tw, "DOMAIN")
		for _, item := range items {
			fmt.Fprintln(tw, item.Hostname)
		}
	} else {
		fmt.Fprintln(tw, "DOMAIN\tAPP")
		for _, item := range items {
			fmt.Fprintf(tw, "%s\t%s\n", item.Hostname, item.App)
		}
	}
	return tw.Flush()
}
