package cli

import (
	"context"

	"github.com/FacileStudio/gare/apps/api/internal/appconfig"
	"github.com/FacileStudio/gare/apps/api/internal/podman"
	"github.com/FacileStudio/gare/apps/api/internal/repopath"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
)

// writeWorkloadArtifacts writes the workload artifacts for the resolved type.
func writeWorkloadArtifacts(ctx context.Context, store *storage.Store, cfg *storage.AppConfig) error {
	switch {
	case cfg.IsStatic():
		return writeStaticArtifacts(ctx, store, cfg)
	case cfg.IsCompose():
		return writeComposeArtifacts(ctx, store, cfg)
	default:
		return writeAppArtifacts(ctx, store, cfg)
	}
}

// flags returns the create flags in the shape appconfig.Resolve ranks above gare.yml.
func (o appCreateOptions) flags() appconfig.Flags {
	return appconfig.Flags{
		Type:          o.appType,
		ComposeFile:   o.composeFile,
		Containerfile: o.containerfile,
		Context:       o.contextDir,
		StaticDir:     o.staticDir,
		BuildCmd:      o.buildCmd,
		Healthcheck:   o.healthcheck,
		Port:          o.port,
		ContainerPort: o.containerPort,
		Tags:          o.tags,
	}
}

// normalizeCreateConfig settles what only the cloned repository can tell: the compose file a compose
// workload runs, and the port a container workload's Containerfile exposes.
func normalizeCreateConfig(repoDir string, cfg *storage.AppConfig) error {
	if cfg.IsCompose() {
		composeFile, err := resolveComposeWorkload(repoDir, cfg.ComposeFile, cfg.Port)
		if err != nil {
			return err
		}
		cfg.ComposeFile = composeFile
	}
	warnStrayComposeFile(repoDir, cfg.IsCompose())
	if cfg.UsesPodManifest() && cfg.ContainerPort == 0 {
		cfg.ContainerPort = podman.DetectExposedPort(repoDir, cfg.Containerfile)
	}
	return nil
}

// validateCreatePaths rejects a --static, --context or --containerfile flag that would reach
// outside the repository, before anything is cloned.
func validateCreatePaths(opts appCreateOptions) error {
	return repopath.ValidateSet(map[string]string{
		"--static":        opts.staticDir,
		"--context":       opts.contextDir,
		"--containerfile": opts.containerfile,
	})
}
