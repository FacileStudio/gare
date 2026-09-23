package main

import (
	"context"
	"fmt"
	"io"
	"sort"
	"text/tabwriter"

	"github.com/FacileStudio/gare/internal/storage"
)

type domainItem struct {
	Hostname string `json:"hostname"`
	App      string `json:"app,omitempty"`
}

func runDomainAdd(ctx context.Context, appName, hostname string) error {
	if err := storage.ValidateAppName(appName); err != nil {
		return err
	}
	baseDir := storage.DefaultBaseDir()
	appDir := storage.GetAppDir(baseDir, appName)
	cfg, err := storage.LoadConfig(appDir)
	if err != nil {
		return appConfigError(appName, err)
	}
	if err := checkDomainAvailable(baseDir, appName, hostname); err != nil {
		return err
	}
	if err := cfg.AddDomain(hostname); err != nil {
		return err
	}
	if err := storage.SaveConfig(appDir, cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	printSuccess(fmt.Sprintf("Added domain %q to app %q", storage.NormalizeDomain(hostname), appName))
	return syncAndReloadIfActive(ctx, cfg)
}

func checkDomainAvailable(baseDir, appName, hostname string) error {
	if err := storage.ValidateDomain(hostname); err != nil {
		return err
	}
	apps, err := storage.ListApps(baseDir)
	if err != nil {
		return fmt.Errorf("failed to list apps: %w", err)
	}
	if owner := storage.AppByDomain(apps, hostname); owner != nil {
		if owner.Name == appName {
			return fmt.Errorf("app %q already owns domain %q", appName, storage.NormalizeDomain(hostname))
		}
		return fmt.Errorf("domain %q is already in use by app %q", storage.NormalizeDomain(hostname), owner.Name)
	}
	return nil
}

func runDomainRemove(ctx context.Context, appName, hostname string) error {
	if err := storage.ValidateAppName(appName); err != nil {
		return err
	}
	baseDir := storage.DefaultBaseDir()
	appDir := storage.GetAppDir(baseDir, appName)
	cfg, err := storage.LoadConfig(appDir)
	if err != nil {
		return appConfigError(appName, err)
	}
	if err := cfg.RemoveDomain(hostname); err != nil {
		return err
	}
	if err := storage.SaveConfig(appDir, cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	printSuccess(fmt.Sprintf("Removed domain %q from app %q", storage.NormalizeDomain(hostname), appName))
	return syncAndReloadIfActive(ctx, cfg)
}

func runDomainList(w io.Writer, targetApp string, opts domainListOptions) error {
	baseDir := storage.DefaultBaseDir()
	var items []domainItem
	if targetApp != "" {
		if err := storage.ValidateAppName(targetApp); err != nil {
			return err
		}
		cfg, err := storage.LoadConfig(storage.GetAppDir(baseDir, targetApp))
		if err != nil {
			return appConfigError(targetApp, err)
		}
		items = collectAppDomains(cfg)
	} else {
		apps, err := storage.ListApps(baseDir)
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
