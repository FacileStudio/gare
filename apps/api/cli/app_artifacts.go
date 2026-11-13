package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/FacileStudio/gare/apps/api/internal/manifest"
	"github.com/FacileStudio/gare/apps/api/internal/repopath"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
)

func writeAppArtifacts(ctx context.Context, store *storage.Store, cfg *storage.AppConfig) error {
	appDir := store.AppDir(cfg.Name)
	if err := resolveManifest(appDir, cfg); err != nil {
		return err
	}
	if err := writeContainerUnit(ctx, cfg.Name, appDir); err != nil {
		return err
	}
	return saveAppMetadata(store, cfg)
}

func writeStaticArtifacts(ctx context.Context, store *storage.Store, cfg *storage.AppConfig) error {
	appDir := store.AppDir(cfg.Name)
	staticPath, err := repopath.Join("static_dir", storage.GetRepoDir(appDir), cfg.StaticDir)
	if err != nil {
		return err
	}
	if err := writeStaticUnit(ctx, cfg.Name, appDir, staticPath, cfg.Port); err != nil {
		return err
	}
	return saveAppMetadata(store, cfg)
}

// resolveManifest adopts the repository's own manifest when it ships one, and otherwise generates a
// default. Adoption goes through the same path a deploy uses, so a manifest gare adopts at create
// time keeps the environment a later deploy would have preserved.
func resolveManifest(appDir string, cfg *storage.AppConfig) error {
	repoDir := storage.GetRepoDir(appDir)
	if _, err := os.Stat(filepath.Join(repoDir, "manifest.yaml")); err == nil {
		return syncRepoManifest(appDir, repoDir)
	}
	containerPort := cfg.ContainerPort
	if containerPort <= 0 {
		containerPort = cfg.Port
	}
	return manifest.Generate(cfg.Name, containerPort, cfg.Port, storage.GetManifestPath(appDir))
}

// saveAppMetadata records a newly created app, stamped with its creation time.
func saveAppMetadata(store *storage.Store, cfg *storage.AppConfig) error {
	cfg.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := store.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	printSuccess(fmt.Sprintf("App %q successfully created on port %d", cfg.Name, cfg.Port))
	return nil
}
