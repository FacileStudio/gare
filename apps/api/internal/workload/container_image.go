package workload

import (
	"context"
	"fmt"

	"github.com/FacileStudio/gare/apps/api/internal/deploys"
	"github.com/FacileStudio/gare/apps/api/internal/git"
	"github.com/FacileStudio/gare/apps/api/internal/manifest"
	"github.com/FacileStudio/gare/apps/api/internal/podman"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
)

// ImageRepo is the local repository every image of an app is tagged in.
func ImageRepo(name string) string {
	return "localhost/" + name
}

// ImageTag is the tag one build of an app is kept under, named after the commit it was built from,
// so an earlier build stays addressable after the next one takes :latest.
func ImageTag(name, commit string) string {
	return fmt.Sprintf("%s:%s", ImageRepo(name), deploys.ShortSHA(commit))
}

// CanRollBack reports whether an app's workload type keeps the images of earlier deploys. Only a
// container workload does: a static site and a compose stack are rebuilt from the checkout alone.
func CanRollBack(cfg *storage.AppConfig) bool {
	workloadType, err := cfg.ResolveWorkloadType()
	return err == nil && workloadType == storage.WorkloadContainer
}

// PointImage makes an app run the image an earlier deploy of commit built: the manifest names its
// tag and :latest moves to it, so a later start without a deploy runs the same image.
func PointImage(ctx context.Context, app App, commit string) error {
	if err := pinImage(app, commit); err != nil {
		return err
	}
	return podman.Tag(ctx, ImageTag(app.Config.Name, commit), ImageRef(app.Config.Name))
}

// revision returns the commit the checkout is at, or "" when git cannot tell, in which case the
// build keeps only :latest and the deploy simply cannot be rolled back to.
func revision(ctx context.Context, app App) string {
	commit, err := git.GetCommitHash(ctx, app.RepoDir())
	if err != nil {
		return ""
	}
	return commit
}

// tagBuild names the image just built as :latest after its commit.
func tagBuild(ctx context.Context, app App, commit string) error {
	if commit == "" {
		return nil
	}
	name := app.Config.Name
	if err := podman.Tag(ctx, ImageRef(name), ImageTag(name, commit)); err != nil {
		return fmt.Errorf("failed to tag the image of %s: %w", name, err)
	}
	return nil
}

// pinImage points the pod manifest at the image built from commit, so the pod runs that build and
// not whatever :latest names at the time it starts.
func pinImage(app App, commit string) error {
	if commit == "" {
		return nil
	}
	name := app.Config.Name
	if err := manifest.SetImage(storage.GetManifestPath(app.Dir), ImageRepo(name), ImageTag(name, commit)); err != nil {
		return fmt.Errorf("failed to point the manifest at %s: %w", ImageTag(name, commit), err)
	}
	return nil
}

// removeImages removes every tag of the app's image, :latest and each kept build, so a destroy
// leaves none of them behind.
func removeImages(ctx context.Context, app App) {
	repo := ImageRepo(app.Config.Name)
	tags, err := podman.ImageTags(ctx, repo)
	if err != nil || len(tags) == 0 {
		tags = []string{"latest"}
	}
	for _, tag := range tags {
		if tag == "<none>" {
			continue
		}
		ref := repo + ":" + tag
		app.Report.Step(fmt.Sprintf("Removing container image %s...", ref))
		bestEffort(app, "removing container image", func() error { return podman.RemoveImage(ctx, ref) })
	}
}
