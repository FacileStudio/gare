package main

import (
	"fmt"
	"os"

	"github.com/FacileStudio/gare/internal/caddy"
	"github.com/FacileStudio/gare/internal/quadlet"
	"github.com/FacileStudio/gare/internal/storage"
	"github.com/FacileStudio/gare/internal/systemd"
)

// writeContainerUnit writes the Quadlet source supervising the application pod manifest.
func writeContainerUnit(name, appDir string) error {
	if err := prepareQuadletUnit(name); err != nil {
		return err
	}
	if err := quadlet.WriteKubeUnit(name, storage.GetManifestPath(appDir)); err != nil {
		return fmt.Errorf("failed to write quadlet unit: %w", err)
	}
	return nil
}

// writeStaticUnit writes the Quadlet source serving the application static directory.
func writeStaticUnit(name, appDir, rootDir string, port int) error {
	if err := prepareQuadletUnit(name); err != nil {
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
	return nil
}

func prepareQuadletUnit(name string) error {
	if err := ensureUnitPathFree(name); err != nil {
		return err
	}
	return quadlet.CheckGenerator()
}

// ensureUnitPathFree rejects a unit file gare does not own, because it would shadow the generated unit.
func ensureUnitPathFree(name string) error {
	unitPath := systemd.GetUnitPath(name)
	if exists, err := pathExists(unitPath); err != nil || !exists {
		return err
	}
	return fmt.Errorf("unit file %s shadows the quadlet unit — remove it and deploy again, "+
		"or run `gare destroy %s` to remove the whole app", unitPath, name)
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

// appUnitExists reports whether gare wrote workload artifacts for an application.
func appUnitExists(name string) (bool, error) {
	if exists, err := pathExists(systemd.GetUnitPath(name)); err != nil || exists {
		return exists, err
	}
	if exists, err := pathExists(quadlet.KubePath(name)); err != nil || exists {
		return exists, err
	}
	return pathExists(quadlet.ContainerPath(name))
}

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
