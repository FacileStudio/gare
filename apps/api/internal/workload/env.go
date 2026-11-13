package workload

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/FacileStudio/gare/apps/api/internal/atomicfile"
	"github.com/FacileStudio/gare/apps/api/internal/manifest"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
	"github.com/FacileStudio/gare/apps/api/internal/systemd"
)

// ReadEnv returns an app's environment from its env file, the one place every workload type keeps
// it. A container app an older gare left with its variables in the manifest reports those until
// its next locked operation moves them, so a read never shows an app with less than it runs with.
func ReadEnv(app App) (map[string]string, error) {
	envs, err := storage.GetAppEnv(app.Dir)
	if err != nil || len(envs) > 0 {
		return envs, err
	}
	pending, err := manifestOnlyEnv(app)
	if err != nil || len(pending) == 0 {
		return envs, err
	}
	return pending, nil
}

// MigrateEnv moves the variables an older gare kept in a container app's manifest into its env
// file, once: it acts only while the env file is empty, and a later render rewrites the manifest
// from the file. The caller holds the app's lock.
func MigrateEnv(app App) error {
	pending, err := manifestOnlyEnv(app)
	if err != nil || len(pending) == 0 {
		return err
	}
	if err := storage.SetAppEnv(app.Dir, pending); err != nil {
		return fmt.Errorf("failed to move the manifest environment into the app env file: %w", err)
	}
	app.Report.Done(fmt.Sprintf("Moved %d environment variable(s) from manifest.yaml to the app env file", len(pending)))
	return nil
}

// manifestOnlyEnv returns what a container app's manifest sets that its env file does not hold yet,
// leaving out entries the repository's own manifest declares with the same value, since those
// arrive with every deploy and freezing them in the env file would pin them against the repository.
func manifestOnlyEnv(app App) (map[string]string, error) {
	workloadType, err := app.Config.ResolveWorkloadType()
	if err != nil || workloadType != storage.WorkloadContainer {
		return nil, err
	}
	current, err := storage.GetAppEnv(app.Dir)
	if err != nil || len(current) > 0 {
		return nil, err
	}
	held, err := manifest.GetEnv(storage.GetManifestPath(app.Dir))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read the manifest environment: %w", err)
	}
	declared, _ := manifest.GetEnv(repoManifestPath(app))
	for k, v := range declared {
		if held[k] == v {
			delete(held, k)
		}
	}
	return held, nil
}

// repoManifestPath returns the manifest the repository ships.
func repoManifestPath(app App) string {
	return filepath.Join(app.RepoDir(), "manifest.yaml")
}

// renderManifestEnv writes the env file into the pod manifest, on top of what the repository's
// manifest declares. kube play passes nothing of the unit's own environment into the pod, so the
// manifest is the only way a container sees the variables.
func renderManifestEnv(app App) error {
	envs, err := storage.GetAppEnv(app.Dir)
	if err != nil {
		return fmt.Errorf("failed to read the app env file: %w", err)
	}
	if err := manifest.RenderEnv(storage.GetManifestPath(app.Dir), repoManifestPath(app), envs); err != nil {
		return fmt.Errorf("failed to write the environment into the manifest: %w", err)
	}
	return nil
}

// writePodmanEnv writes the file a static site's `podman run --env-file` reads. podman reads an
// env file literally, quotes and all, and cannot express a multi-line value, so it cannot read the
// app env file systemd reads. This file names each variable without a value instead, which tells
// podman to take it from its own environment, where the unit's EnvironmentFile= put it after
// systemd parsed the app env file.
func writePodmanEnv(app App) error {
	envs, err := storage.GetAppEnv(app.Dir)
	if err != nil {
		return fmt.Errorf("failed to read the app env file: %w", err)
	}
	var content strings.Builder
	for _, k := range slices.Sorted(maps.Keys(envs)) {
		content.WriteString(k + "\n")
	}
	path := systemd.PodmanEnvFile(storage.GetAppEnvPath(app.Dir))
	if err := atomicfile.WriteFile(path, []byte(content.String()), 0600); err != nil {
		return fmt.Errorf("failed to write the podman env file: %w", err)
	}
	return nil
}

// manifestExists reports whether gare's copy of the pod manifest is on disk yet.
func manifestExists(app App) bool {
	_, err := os.Stat(storage.GetManifestPath(app.Dir))
	return err == nil
}
