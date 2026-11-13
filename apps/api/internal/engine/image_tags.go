package engine

import (
	"context"
	"fmt"
	"regexp"
	"slices"

	"github.com/FacileStudio/gare/apps/api/internal/deploys"
	"github.com/FacileStudio/gare/apps/api/internal/podman"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
	"github.com/FacileStudio/gare/apps/api/internal/workload"
)

// keptImageTags is how many commit-tagged images of an app are kept to roll back to.
const keptImageTags = 5

// pruneImageTags removes the commit tags of a container app beyond the builds of its newest
// keptImageTags deployed commits. Like image pruning it is best-effort and reported only in verbose
// mode, since a tag left behind costs disk and says nothing about the deploy.
func (e *Engine) pruneImageTags(ctx context.Context, cfg *storage.AppConfig) {
	if !workload.CanRollBack(cfg) {
		return
	}
	report := e.report(cfg.Name)
	repo := workload.ImageRepo(cfg.Name)
	tags, err := podman.ImageTags(ctx, repo)
	if err != nil {
		report.Verbose(fmt.Sprintf("Skipped image tag pruning: %v", err))
		return
	}
	list, err := deploys.Open(e.store.AppDir(cfg.Name), cfg.Name).List()
	if err != nil {
		report.Verbose(fmt.Sprintf("Skipped image tag pruning: %v", err))
		return
	}
	for _, tag := range staleTags(tags, keepTags(list)) {
		if err := podman.RemoveImage(ctx, repo+":"+tag); err != nil {
			report.Verbose(fmt.Sprintf("Could not remove image %s:%s: %v", repo, tag, err))
		}
	}
}

// keepTags returns the commit tags of the newest keptImageTags distinct commits in a history listed
// newest first, the current deploy's included.
func keepTags(list []deploys.Deploy) []string {
	var keep []string
	for _, d := range list {
		tag := deploys.ShortSHA(d.Commit)
		if tag == "" || slices.Contains(keep, tag) {
			continue
		}
		keep = append(keep, tag)
		if len(keep) == keptImageTags {
			break
		}
	}
	return keep
}

// staleTags returns the commit tags not in keep. Only a tag shaped like a commit is a candidate,
// so :latest and any tag an operator added by hand survive.
func staleTags(tags, keep []string) []string {
	commitTag := regexp.MustCompile(`^[0-9a-f]{7,40}$`)
	var stale []string
	for _, tag := range tags {
		if commitTag.MatchString(tag) && !slices.Contains(keep, tag) {
			stale = append(stale, tag)
		}
	}
	return stale
}
