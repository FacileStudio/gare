package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/FacileStudio/gare/internal/atomicfile"
	"github.com/FacileStudio/gare/internal/storage"
)

func syncManifest(name, appDir, repoDir string, cfg *storage.AppConfig) error {
	repoManifest := filepath.Join(repoDir, "manifest.yaml")
	if _, err := os.Stat(repoManifest); err == nil {
		return syncRepoManifest(appDir, repoDir)
	}
	manifestPath := storage.GetManifestPath(appDir)
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		return storage.GenerateDefaultManifest(name, cfg.ContainerPort, cfg.Port, manifestPath)
	}
	cPort := cfg.ContainerPort
	if cPort <= 0 {
		cPort = cfg.Port
	}
	return storage.UpdateManifestPorts(manifestPath, cPort, cfg.Port)
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
	existingEnvs, _ := storage.GetManifestEnv(appManifest)
	if err := atomicfile.WriteFile(appManifest, data, 0644); err != nil {
		return fmt.Errorf("failed to sync manifest: %w", err)
	}
	if len(existingEnvs) > 0 {
		if err := storage.SetManifestEnv(appManifest, existingEnvs); err != nil {
			return fmt.Errorf("failed to restore manifest environment: %w", err)
		}
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
