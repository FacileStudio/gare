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

// envStore is where an application's environment variables actually live: the app env file a
// static or compose workload reads, or the pod manifest a container workload reads. Its operations
// are fields with distinct names, so an app env writer cannot be passed where a manifest writer
// belongs — the mix-up two identically typed closures invite.
type envStore struct {
	label  string
	read   func(appDir string) (map[string]string, error)
	write  func(appDir string, vars map[string]string) error
	remove func(appDir string, keys []string) error
}

// appEnvFileStore returns the store a static or compose workload reads.
func appEnvFileStore() envStore {
	return envStore{
		label:  "app env file",
		read:   storage.GetAppEnv,
		write:  storage.SetAppEnv,
		remove: storage.UnsetAppEnv,
	}
}

// podManifestStore returns the store a container workload reads.
func podManifestStore() envStore {
	return envStore{
		label: "manifest",
		read: func(appDir string) (map[string]string, error) {
			return manifest.GetEnv(storage.GetManifestPath(appDir))
		},
		write: func(appDir string, vars map[string]string) error {
			return manifest.SetEnv(storage.GetManifestPath(appDir), vars)
		},
		remove: func(appDir string, keys []string) error {
			return manifest.UnsetEnv(storage.GetManifestPath(appDir), keys)
		},
	}
}

// resolveEnvStore resolves a managed application and the store its workload reads, so the
// app-env-file-or-manifest choice is made in one place and every command lands where the workload
// looks.
func resolveEnvStore(name string) (string, envStore, error) {
	appDir, cfg, err := loadAppConfig(name)
	if err != nil {
		return "", envStore{}, err
	}
	if cfg.UsesAppEnvFile() {
		return appDir, appEnvFileStore(), nil
	}
	return appDir, podManifestStore(), nil
}

// envStoreError names the store a failed environment change was aimed at, so a failure says which
// of the two stores could not be written.
func envStoreError(action string, store envStore, err error) error {
	return fmt.Errorf("failed to %s environment variables in the %s: %w", action, store.label, err)
}

func runEnvList(w io.Writer, name string, opts envListOptions) error {
	appDir, store, err := resolveEnvStore(name)
	if err != nil {
		return err
	}
	envs, err := store.read(appDir)
	if err != nil {
		return fmt.Errorf("failed to read environment variables from the %s: %w", store.label, err)
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
