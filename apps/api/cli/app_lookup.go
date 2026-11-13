package cli

import (
	"github.com/FacileStudio/gare/apps/api/internal/appname"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
)

// loadAppConfig resolves a managed application's storage directory and configuration, so every
// command that operates on one application agrees on validation and on the error an unknown app
// reports.
func loadAppConfig(name string) (string, *storage.AppConfig, error) {
	if err := appname.Validate(name); err != nil {
		return "", nil, err
	}
	store := storage.Default()
	cfg, err := store.Load(name)
	if err != nil {
		return "", nil, appConfigError(name, err)
	}
	return store.AppDir(name), cfg, nil
}
