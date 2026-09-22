package main

import (
	"context"
	"fmt"
	"os"

	"github.com/FacileStudio/gare/internal/quadlet"
	"github.com/FacileStudio/gare/internal/systemd"
)

// appUnitExists reports whether gare wrote workload artifacts for an application.
func appUnitExists(name string) (bool, error) {
	if exists, err := pathExists(systemd.GetUnitPath(name)); err != nil || exists {
		return exists, err
	}
	return quadletSourcesExist(name)
}

// quadletSourcesExist reports whether a Quadlet source file gare owns exists for an application.
func quadletSourcesExist(name string) (bool, error) {
	for _, path := range []string{quadlet.KubePath(name), quadlet.ContainerPath(name)} {
		exists, err := pathExists(path)
		if err != nil || exists {
			return exists, err
		}
	}
	return false, nil
}

// stopQuadletWorkload stops a workload the Quadlet generator supervises, so the generated unit's
// own teardown runs before a synthesized unit takes over the same name.
func stopQuadletWorkload(ctx context.Context, name string) error {
	exists, err := quadletSourcesExist(name)
	if err != nil || !exists {
		return err
	}
	printInfo(fmt.Sprintf("Stopping the Quadlet workload for %s before the synthesized unit takes over...", name))
	if err := systemd.Stop(ctx, name); err != nil {
		printWarning(fmt.Sprintf("systemctl stop returned error: %v", err))
	}
	return nil
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
