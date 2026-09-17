package main

import (
	"os"
	"path/filepath"

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

func defaultStr(val, fallback string) string {
	if val != "" {
		return val
	}
	return fallback
}
