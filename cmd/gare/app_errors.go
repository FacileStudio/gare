package main

import (
	"fmt"

	"github.com/FacileStudio/gare/internal/storage"
)

// appConfigError wraps a failed application config lookup with a remedy.
func appConfigError(name string, err error) error {
	return fmt.Errorf("app %q not found — run `gare list` to see managed apps: %w", name, err)
}

// loadAppConfig resolves a managed application's storage directory and configuration, so every
// command that operates on one application agrees on validation and on the error an unknown app
// reports.
func loadAppConfig(name string) (string, *storage.AppConfig, error) {
	if err := storage.ValidateAppName(name); err != nil {
		return "", nil, err
	}
	appDir := storage.GetAppDir(storage.DefaultBaseDir(), name)
	cfg, err := storage.LoadConfig(appDir)
	if err != nil {
		return "", nil, appConfigError(name, err)
	}
	return appDir, cfg, nil
}
