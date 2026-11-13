package storage

import (
	"path/filepath"
)

// GetRepoDir returns the repository directory path within an app directory.
func GetRepoDir(appDir string) string {
	return filepath.Join(appDir, "repo")
}

// GetManifestPath returns the manifest.yaml path within an app directory.
func GetManifestPath(appDir string) string {
	return filepath.Join(appDir, "manifest.yaml")
}
