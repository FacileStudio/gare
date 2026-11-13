package cli

import (
	"fmt"
	"path/filepath"

	"github.com/FacileStudio/gare/apps/api/internal/appconfig"
	"github.com/FacileStudio/gare/apps/api/internal/compose"
	"github.com/FacileStudio/gare/apps/api/internal/health"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
)

// resolveDeployConfig re-reads gare.yml after a sync and settles cfg by the precedence
// appconfig.Resolve applies, so a value changed in gare.yml takes effect at this deploy. A port
// gare.yml moves to must be free, and a compose workload whose gare.yml declares no probes gets the
// ones its compose file implies, derived afresh every deploy.
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
	if resolved.IsCompose() && (gf == nil || len(gf.Healthchecks) == 0) {
		resolved.SetHealth(resolved.HealthPath(), deriveComposeProbes(repoDir, resolved))
	}
	*cfg = *resolved
	return nil
}

func deriveComposeProbes(repoDir string, cfg *storage.AppConfig) []health.Probe {
	composeFile, err := compose.LocateFile(repoDir, cfg.ComposeFile)
	if err != nil {
		return nil
	}
	probes, err := compose.HealthProbes(filepath.Join(repoDir, composeFile))
	if err != nil || len(probes) == 0 {
		return nil
	}
	printInfo(fmt.Sprintf("Derived %d health probe(s) from %s: %s", len(probes), composeFile, probeSummary(probes)))
	return probes
}
