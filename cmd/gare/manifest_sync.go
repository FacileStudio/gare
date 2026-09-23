package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/FacileStudio/gare/internal/atomicfile"
	"github.com/FacileStudio/gare/internal/manifest"
	"github.com/FacileStudio/gare/internal/storage"
)

func syncManifest(name, appDir, repoDir string, cfg *storage.AppConfig) error {
	if err := writeManifest(name, appDir, repoDir, cfg); err != nil {
		return err
	}
	return manifest.EnsureLocalImagePullPolicy(storage.GetManifestPath(appDir))
}

// writeManifest resolves the manifest gare supervises, adopting a repository's own file when it
// ships one and otherwise generating or updating gare's copy.
func writeManifest(name, appDir, repoDir string, cfg *storage.AppConfig) error {
	repoManifest := filepath.Join(repoDir, "manifest.yaml")
	if _, err := os.Stat(repoManifest); err == nil {
		return syncRepoManifest(appDir, repoDir)
	}
	manifestPath := storage.GetManifestPath(appDir)
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		return manifest.Generate(name, cfg.ContainerPort, cfg.Port, manifestPath)
	}
	cPort := cfg.ContainerPort
	if cPort <= 0 {
		cPort = cfg.Port
	}
	return manifest.UpdatePorts(manifestPath, cPort, cfg.Port)
}

func syncRepoManifest(appDir, repoDir string) error {
	repoManifest := filepath.Join(repoDir, "manifest.yaml")
	data, err := os.ReadFile(repoManifest)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read repo manifest: %w", err)
	}
	appManifest := storage.GetManifestPath(appDir)
	existingEnvs, _ := manifest.GetEnv(appManifest)
	if err := atomicfile.WriteFile(appManifest, data, 0644); err != nil {
		return fmt.Errorf("failed to sync manifest: %w", err)
	}
	if len(existingEnvs) > 0 {
		if err := manifest.SetEnv(appManifest, existingEnvs); err != nil {
			return fmt.Errorf("failed to restore manifest environment: %w", err)
		}
	}
	if err := manifest.EnsureLocalImagePullPolicy(appManifest); err != nil {
		return fmt.Errorf("failed to pin the local image pull policy: %w", err)
	}
	printSuccess("Synced manifest.yaml from repository")
	return nil
}

func defaultStr(val, fallback string) string {
	if val != "" {
		return val
	}
	return fallback
}
