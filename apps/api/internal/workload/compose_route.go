package workload

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/FacileStudio/gare/apps/api/internal/compose"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
)

// locateCompose resolves the compose file and validates how gare routes to the stack: through the
// override file when gare.yml names a service, or else through a port the compose file publishes
// itself. Nothing is written before both checks pass.
func locateCompose(app App) (string, error) {
	cfg := app.Config
	if cfg.Service == "" {
		return resolveComposeFile(app.RepoDir(), cfg.ComposeFile, cfg.Port)
	}
	composeFile, err := compose.LocateFile(app.RepoDir(), cfg.ComposeFile)
	if err != nil {
		return "", err
	}
	composePath := filepath.Join(app.RepoDir(), composeFile)
	return composeFile, compose.ValidateService(composePath, cfg.Service, cfg.ContainerPort)
}

// routedPublishes returns the port gare publishes for the routed service, and none for an app whose
// compose file publishes the routed port itself.
func routedPublishes(cfg *storage.AppConfig) []compose.Publish {
	if cfg.Service == "" {
		return nil
	}
	return []compose.Publish{{Service: cfg.Service, HostPort: cfg.Port, ContainerPort: cfg.ContainerPort}}
}

// overridePath returns where an app's compose override file lives, beside its config rather than in
// the checkout, so a repository sync never touches it.
func overridePath(app App) string {
	return filepath.Join(app.Dir, compose.OverrideFileName)
}

// assignComposePort gives a service-routed app stored without a port a free one at deploy, the way
// a container app gets one, since the host port is gare's to choose once the override publishes it.
func assignComposePort(app App) error {
	cfg := app.Config
	if cfg.Service == "" || cfg.Port != 0 {
		return nil
	}
	port, err := app.Store.AvailablePort(0)
	if err != nil {
		return fmt.Errorf("failed to discover free port: %w", err)
	}
	cfg.Port = port
	return nil
}

// syncOverride writes the override file of a service-routed app and removes a stale one from an
// app that no longer names a service, so the unit and the file on disk always agree.
func syncOverride(app App) error {
	path := overridePath(app)
	publishes := routedPublishes(app.Config)
	if len(publishes) == 0 {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	return compose.WriteOverride(path, publishes[0])
}

// unitOverride returns the override file a unit or a compose call passes after the repository's
// compose file, or empty when the app routes to a port its compose file publishes.
func unitOverride(app App) string {
	if app.Config.Service == "" {
		return ""
	}
	return overridePath(app)
}

// teardownOverride returns the override file a teardown passes when it is still on disk, so down
// sees the stack up started without failing on a file a previous destroy already removed.
func teardownOverride(app App) string {
	path := overridePath(app)
	if _, err := os.Stat(path); err != nil {
		return ""
	}
	return path
}
