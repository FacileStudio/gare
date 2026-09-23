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

// applyEnvChange writes environment changes through the store the workload actually reads: the app
// env file for static and compose workloads, the pod manifest otherwise. It returns the store's name
// so the caller can say where the change landed.
func applyEnvChange(name string, appEnv func(appDir string) error, podEnv func(manifestPath string) error) (string, error) {
	appDir, cfg, err := loadAppConfig(name)
	if err != nil {
		return "", err
	}
	if cfg.UsesAppEnvFile() {
		if err := appEnv(appDir); err != nil {
			return "", err
		}
		return "app env file", nil
	}
	if err := podEnv(storage.GetManifestPath(appDir)); err != nil {
		return "", err
	}
	return "manifest", nil
}

func runEnvList(w io.Writer, name string, opts envListOptions) error {
	appDir, cfg, err := loadAppConfig(name)
	if err != nil {
		return err
	}
	var envs map[string]string
	if cfg.UsesAppEnvFile() {
		envs, err = storage.GetAppEnv(appDir)
	} else {
		envs, err = manifest.GetEnv(storage.GetManifestPath(appDir))
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
