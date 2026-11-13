package engine

import (
	"fmt"

	"github.com/FacileStudio/gare/apps/api/internal/appconfig"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
)

// resolveDeployConfig re-reads gare.yml after a sync and settles cfg by the precedence
// appconfig.Resolve applies, so a value changed in gare.yml takes effect at this deploy. A port
// gare.yml moves to must be free; a refused port leaves cfg as it was.
func resolveDeployConfig(store *storage.Store, repoDir string, cfg *storage.AppConfig) error {
	gf, err := storage.LoadGareFile(repoDir)
	if err != nil {
		return err
	}
	resolved, err := appconfig.Resolve(appconfig.Flags{}, gf, cfg)
	if err != nil {
		return err
	}
	if resolved.Port > 0 && resolved.Port != cfg.Port {
		if _, err := store.AvailablePort(resolved.Port); err != nil {
			return fmt.Errorf("port %d in gare.yml is not available — choose another port: %w", resolved.Port, err)
		}
	}
	*cfg = *resolved
	return nil
}
