package storage

import "path/filepath"

// GetAppStaticConfigPath returns the Caddyfile path a static workload's container serves.
func GetAppStaticConfigPath(appDir string) string {
	return filepath.Join(appDir, "Caddyfile")
}
