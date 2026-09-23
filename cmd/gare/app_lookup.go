package main

import (
	"github.com/FacileStudio/gare/internal/storage"
)

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
