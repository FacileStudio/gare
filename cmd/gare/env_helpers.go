package main

import (
	"context"
	"fmt"
	"io"
	"sort"

	"github.com/FacileStudio/gare/internal/manifest"
	"github.com/FacileStudio/gare/internal/storage"
	"github.com/FacileStudio/gare/internal/systemd"
)

func runEnvList(w io.Writer, name string, opts envListOptions) error {
	manifestPath, cfg, err := getAppManifestInfo(name)
	if err != nil {
		return err
	}
	var envs map[string]string
	if cfg.UsesAppEnvFile() {
		appDir := storage.GetAppDir(storage.DefaultBaseDir(), name)
		envs, err = storage.GetAppEnv(appDir)
	} else {
		envs, err = manifest.GetEnv(manifestPath)
	}
	if err != nil {
		return fmt.Errorf("failed to read environment variables: %w", err)
	}
	if opts.jsonOutput {
		return writeJSON(w, envs)
	}
	return outputEnvPlain(w, envs)
}

func outputEnvPlain(w io.Writer, envs map[string]string) error {
	if len(envs) == 0 {
		fmt.Fprintln(w, "No environment variables configured")
		return nil
	}
	keys := make([]string, 0, len(envs))
	for k := range envs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(w, "%s=%s\n", k, envs[k])
	}
	return nil
}

func getAppManifestInfo(name string) (string, *storage.AppConfig, error) {
	if err := storage.ValidateAppName(name); err != nil {
		return "", nil, err
	}
	baseDir := storage.DefaultBaseDir()
	appDir := storage.GetAppDir(baseDir, name)
	cfg, err := storage.LoadConfig(appDir)
	if err != nil {
		return "", nil, appConfigError(name, err)
	}
	manifestPath := storage.GetManifestPath(appDir)
	return manifestPath, cfg, nil
}

func reloadIfActive(ctx context.Context, name string) error {
	status, err := systemd.IsActive(ctx, name)
	if err == nil && status == "active" {
		printInfo(fmt.Sprintf("Restarting %s.service to apply environment changes...", name))
		if err := systemd.Restart(ctx, name); err != nil {
			return fmt.Errorf("failed to restart %s: %w", name, err)
		}
		if err := systemd.WaitForState(ctx, name, "active"); err != nil {
			return fmt.Errorf("failed to verify %s active state: %w", name, err)
		}
		printSuccess(fmt.Sprintf("Restarted %s.service (active)", name))
		return nil
	}
	printInfo(fmt.Sprintf("Service %s is not active (%s); environment saved", name, status))
	return nil
}
