package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/FacileStudio/gare/internal/caddy"
	"github.com/FacileStudio/gare/internal/quadlet"
	"github.com/FacileStudio/gare/internal/storage"
	"github.com/FacileStudio/gare/internal/systemd"
)

// writeContainerUnit writes the Quadlet source supervising the application pod manifest.
func writeContainerUnit(ctx context.Context, name, appDir string) error {
	retire, err := ensureQuadletReady(name)
	if err != nil {
		return err
	}
	if err := quadlet.WriteKubeUnit(name, storage.GetManifestPath(appDir)); err != nil {
		return fmt.Errorf("failed to write quadlet unit: %w", err)
	}
	return retireShadowingUnit(ctx, name, retire)
}

// writeStaticUnit writes the Quadlet source serving the application static directory.
func writeStaticUnit(ctx context.Context, name, appDir, rootDir string, port int) error {
	retire, err := ensureQuadletReady(name)
	if err != nil {
		return err
	}
	if err := storage.EnsureAppEnvFile(appDir); err != nil {
		return fmt.Errorf("failed to prepare the application env file: %w", err)
	}
	data := quadlet.StaticSiteUnit(name, port, rootDir, storage.GetAppEnvPath(appDir), storage.GetAppStaticConfigPath(appDir))
	if err := caddy.WriteStaticServerConfig(data.ConfigFile, data.RootMount, data.ContainerPort); err != nil {
		return fmt.Errorf("failed to write the static site Caddyfile: %w", err)
	}
	if err := quadlet.WriteContainerUnit(data); err != nil {
		return fmt.Errorf("failed to write quadlet unit: %w", err)
	}
	return retireShadowingUnit(ctx, name, retire)
}

// ensureQuadletReady fails before anything is written or stopped when the generated unit could not
// take over, and reports whether a unit file gare itself wrote still occupies the name.
func ensureQuadletReady(name string) (bool, error) {
	if err := quadlet.CheckGenerator(); err != nil {
		return false, err
	}
	unitPath := systemd.GetUnitPath(name)
	exists, err := pathExists(unitPath)
	if err != nil || !exists {
		return false, err
	}
	if !isGareManagedUnit(unitPath, name) {
		return false, fmt.Errorf("unit file %s shadows the quadlet unit — remove it and deploy again, "+
			"or run `gare destroy %s` to remove the whole app", unitPath, name)
	}
	return true, nil
}

// retireShadowingUnit clears the unit file gare wrote before this workload moved to Quadlet. It runs
// after the source is written, so a source that cannot be generated leaves the running workload
// alone, and stops the unit while its own definition is still loaded so its teardown still runs.
func retireShadowingUnit(ctx context.Context, name string, retire bool) error {
	if !retire {
		return nil
	}
	unitPath := systemd.GetUnitPath(name)
	printInfo(fmt.Sprintf("Retiring gare's own unit file at %s before the generated unit takes over...", unitPath))
	if err := systemd.Stop(ctx, name); err != nil {
		printWarning(fmt.Sprintf("systemctl stop returned error: %v", err))
	}
	if err := systemd.RemoveUnit(name); err != nil {
		return fmt.Errorf("failed to retire %s: %w", unitPath, err)
	}
	return nil
}

// isGareManagedUnit reports whether a unit file is one gare synthesized, identified by the
// description each of its workload templates names the application with.
func isGareManagedUnit(unitPath, name string) bool {
	data, err := os.ReadFile(unitPath)
	if err != nil {
		return false
	}
	lines := strings.Split(string(data), "\n")
	return hasUnitLine(lines, "Description=Gare Managed App: "+name) ||
		hasUnitLine(lines, "Description=Gare Managed Static App: "+name) ||
		hasUnitLine(lines, "Description=Gare Managed Compose App: "+name)
}

func hasUnitLine(lines []string, want string) bool {
	for _, line := range lines {
		if strings.TrimSpace(line) == want {
			return true
		}
	}
	return false
}

// checkQuadletGenerator reports whether Podman can generate units from Quadlet sources.
func checkQuadletGenerator() {
	if err := quadlet.CheckGenerator(); err != nil {
		printWarning(err.Error())
		fmt.Printf("\nContainer and static workloads deploy through Quadlet; compose workloads are unaffected.\n\n")
		return
	}
	printSuccess("Found podman quadlet generator")
}
