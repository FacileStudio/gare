package workload

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/FacileStudio/gare/apps/api/internal/atomicfile"
	"github.com/FacileStudio/gare/apps/api/internal/manifest"
	"github.com/FacileStudio/gare/apps/api/internal/podman"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
	"github.com/FacileStudio/gare/apps/api/internal/systemd"
)

// buildImage builds the application image from the configured Containerfile and context as
// :latest, then keeps it under the tag of the commit it was built from.
func buildImage(ctx context.Context, app App, commit string) error {
	cfg := app.Config
	imageName := ImageRef(cfg.Name)
	app.Report.Step(fmt.Sprintf("Building container image %s...", imageName))
	buildOpts := podman.BuildOptions{
		RepoDir:       app.RepoDir(),
		ImageName:     imageName,
		Containerfile: cfg.Containerfile,
		ContextDir:    cfg.ContextDir,
		Stdout:        app.Stdout,
		Stderr:        app.Stderr,
	}
	if err := podman.Build(ctx, buildOpts); err != nil {
		return fmt.Errorf("container build failed: %w", err)
	}
	return tagBuild(ctx, app, commit)
}

// writeContainerUnit writes the systemd unit supervising the application pod manifest.
func writeContainerUnit(ctx context.Context, app App) error {
	name := app.Config.Name
	previous, err := claimUnit(ctx, name, true)
	if err != nil {
		return err
	}
	return handOver(ctx, app, systemd.KubeUnitDescription(name), previous,
		func() error { return systemd.WriteKubeUnit(name, storage.GetManifestPath(app.Dir)) })
}

// resolveManifest adopts the repository's own manifest when it ships one, and otherwise generates a
// default, then writes the app env file into it, the same way a deploy does.
func resolveManifest(app App) error {
	if err := adoptOrGenerateManifest(app); err != nil {
		return err
	}
	return renderManifestEnv(app)
}

// adoptOrGenerateManifest writes gare's copy of the manifest at create time.
func adoptOrGenerateManifest(app App) error {
	cfg := app.Config
	if _, err := os.Stat(filepath.Join(app.RepoDir(), "manifest.yaml")); err == nil {
		return syncRepoManifest(app)
	}
	containerPort := cfg.ContainerPort
	if containerPort <= 0 {
		containerPort = cfg.Port
	}
	return manifest.Generate(cfg.Name, containerPort, cfg.Port, storage.GetManifestPath(app.Dir))
}

// syncManifest resolves the manifest a deploy plays, writes the app env file into it, pins its
// local images to never be pulled and points it at the build of commit. Variables an older gare
// kept in the manifest move to the env file first, before the repository's copy replaces it.
func syncManifest(app App, commit string) error {
	if err := MigrateEnv(app); err != nil {
		return err
	}
	if err := writeManifest(app); err != nil {
		return err
	}
	if err := renderManifestEnv(app); err != nil {
		return err
	}
	if err := manifest.EnsureLocalImagePullPolicy(storage.GetManifestPath(app.Dir)); err != nil {
		return err
	}
	return pinImage(app, commit)
}

// writeManifest resolves the manifest gare supervises, adopting a repository's own file when it
// ships one and otherwise generating or updating gare's copy.
func writeManifest(app App) error {
	cfg := app.Config
	if _, err := os.Stat(filepath.Join(app.RepoDir(), "manifest.yaml")); err == nil {
		return syncRepoManifest(app)
	}
	manifestPath := storage.GetManifestPath(app.Dir)
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		return manifest.Generate(cfg.Name, cfg.ContainerPort, cfg.Port, manifestPath)
	}
	cPort := cfg.ContainerPort
	if cPort <= 0 {
		cPort = cfg.Port
	}
	return manifest.UpdatePorts(manifestPath, cPort, cfg.Port)
}

// syncRepoManifest copies the repository's manifest over gare's copy. The operator's variables live
// in the app env file, which the caller writes into the copy afterwards.
func syncRepoManifest(app App) error {
	data, err := os.ReadFile(filepath.Join(app.RepoDir(), "manifest.yaml"))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read repo manifest: %w", err)
	}
	appManifest := storage.GetManifestPath(app.Dir)
	if err := atomicfile.WriteFile(appManifest, data, 0600); err != nil {
		return fmt.Errorf("failed to sync manifest: %w", err)
	}
	if err := manifest.EnsureLocalImagePullPolicy(appManifest); err != nil {
		return fmt.Errorf("failed to pin the local image pull policy: %w", err)
	}
	app.Report.Done("Synced manifest.yaml from repository")
	return nil
}
