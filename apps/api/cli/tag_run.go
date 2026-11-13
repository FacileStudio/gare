package cli

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/FacileStudio/gare/apps/api/internal/storage"
)

type tagSummaryItem struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}

type appTagItem struct {
	Tag string `json:"tag"`
}

// runTagAdd adds tags to an application through the engine and prints its tags afterwards.
func runTagAdd(ctx context.Context, appName string, tags []string) error {
	updated, err := newEngine(ctx).AddTags(ctx, appName, tags)
	if err != nil {
		return err
	}
	printSuccess(fmt.Sprintf("Updated tags for app %q: %s", appName, strings.Join(updated, ", ")))
	return nil
}

// runTagRemove removes tags from an application through the engine and prints its tags
// afterwards, or a warning when none of them was set.
func runTagRemove(ctx context.Context, appName string, tags []string) error {
	updated, changed, err := newEngine(ctx).RemoveTags(ctx, appName, tags)
	if err != nil {
		return err
	}
	if !changed {
		printWarning(fmt.Sprintf("None of the specified tags were found on app %q", appName))
		return nil
	}
	printSuccess(fmt.Sprintf("Updated tags for app %q: %s", appName, strings.Join(updated, ", ")))
	return nil
}

func runTagList(ctx context.Context, w io.Writer, targetApp string, opts tagListOptions) error {
	if targetApp != "" {
		cfg, err := newEngine(ctx).Config(targetApp)
		if err != nil {
			return err
		}
		return outputAppTags(w, targetApp, cfg.Tags, opts.jsonOutput)
	}
	apps, err := storage.Default().List()
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
		return writeJSON(w, items)
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
		return writeJSON(w, items)
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
