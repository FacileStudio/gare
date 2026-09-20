package main

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/FacileStudio/gare/internal/storage"
)

type tagSummaryItem struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}

type appTagItem struct {
	Tag string `json:"tag"`
}

func runTagAdd(appName string, tags []string) error {
	if err := storage.ValidateAppName(appName); err != nil {
		return err
	}
	baseDir := storage.DefaultBaseDir()
	appDir := storage.GetAppDir(baseDir, appName)
	cfg, err := storage.LoadConfig(appDir)
	if err != nil {
		return appConfigError(appName, err)
	}
	if err := cfg.AddTags(tags...); err != nil {
		return err
	}
	if err := storage.SaveConfig(appDir, cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	printSuccess(fmt.Sprintf("Updated tags for app %q: %s", appName, strings.Join(cfg.Tags, ", ")))
	return nil
}

func runTagRemove(appName string, tags []string) error {
	if err := storage.ValidateAppName(appName); err != nil {
		return err
	}
	baseDir := storage.DefaultBaseDir()
	appDir := storage.GetAppDir(baseDir, appName)
	cfg, err := storage.LoadConfig(appDir)
	if err != nil {
		return appConfigError(appName, err)
	}
	if !cfg.RemoveTags(tags...) {
		printWarning(fmt.Sprintf("None of the specified tags were found on app %q", appName))
		return nil
	}
	if err := storage.SaveConfig(appDir, cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	printSuccess(fmt.Sprintf("Updated tags for app %q: %s", appName, strings.Join(cfg.Tags, ", ")))
	return nil
}

func runTagList(w io.Writer, targetApp string, opts tagListOptions) error {
	baseDir := storage.DefaultBaseDir()
	if targetApp != "" {
		if err := storage.ValidateAppName(targetApp); err != nil {
			return err
		}
		cfg, err := storage.LoadConfig(storage.GetAppDir(baseDir, targetApp))
		if err != nil {
			return appConfigError(targetApp, err)
		}
		return outputAppTags(w, targetApp, cfg.Tags, opts.jsonOutput)
	}
	apps, err := storage.ListApps(baseDir)
	if err != nil {
		return fmt.Errorf("failed to list apps: %w", err)
	}
	return outputAllTags(w, apps, opts.jsonOutput)
}

func outputAppTags(w io.Writer, targetApp string, tags []string, jsonOut bool) error {
	if jsonOut {
		items := make([]appTagItem, 0, len(tags))
		for _, tag := range tags {
			items = append(items, appTagItem{Tag: tag})
		}
		data, err := json.MarshalIndent(items, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(w, string(data))
		return nil
	}
	if len(tags) == 0 {
		fmt.Fprintf(w, "No tags configured for app %q\n", targetApp)
		return nil
	}
	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
	fmt.Fprintln(tw, "TAG")
	for _, tag := range tags {
		fmt.Fprintln(tw, tag)
	}
	return tw.Flush()
}

func outputAllTags(w io.Writer, apps []*storage.AppConfig, jsonOut bool) error {
	counts := storage.CollectAllTags(apps)
	items := make([]tagSummaryItem, 0, len(counts))
	for tag, count := range counts {
		items = append(items, tagSummaryItem{Tag: tag, Count: count})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count != items[j].Count {
			return items[i].Count > items[j].Count
		}
		return items[i].Tag < items[j].Tag
	})
	if jsonOut {
		data, err := json.MarshalIndent(items, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(w, string(data))
		return nil
	}
	if len(items) == 0 {
		fmt.Fprintln(w, "No tags configured")
		return nil
	}
	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
	fmt.Fprintln(tw, "TAG\tAPPS")
	for _, item := range items {
		fmt.Fprintf(tw, "%s\t%d\n", item.Tag, item.Count)
	}
	return tw.Flush()
}
