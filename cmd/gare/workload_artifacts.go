package main

import (
	"os"

	"github.com/FacileStudio/gare/internal/systemd"
)

// appUnitExists reports whether gare wrote workload artifacts for an application.
// Quadlet sources count too, so an application deployed by an older gare is still recognised
// before its next deploy retires them.
func appUnitExists(name string) (bool, error) {
	if exists, err := pathExists(systemd.GetUnitPath(name)); err != nil || exists {
		return exists, err
	}
	return systemd.LegacyQuadletSourcesExist(name)
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
