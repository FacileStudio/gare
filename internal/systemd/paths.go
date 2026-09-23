package systemd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/FacileStudio/gare/internal/xdg"
)

const (
	legacyKubeExtension      = ".kube"
	legacyContainerExtension = ".container"
)

// validateUnitPath rejects paths systemd cannot carry into an ExecStart command line. systemd
// splits the command on whitespace without honouring quoting here, so a space silently becomes
// an extra argument and the unit starts the wrong command.
func validateUnitPath(label, path string) error {
	if path == "" {
		return fmt.Errorf("workload units require a %s path", label)
	}
	if !filepath.IsAbs(path) {
		return fmt.Errorf("workload units require an absolute %s path, got %q", label, path)
	}
	if strings.ContainsAny(path, " \t\n\r\"") {
		return fmt.Errorf("%s path %q contains characters systemd cannot split into a command", label, path)
	}
	return nil
}

// LegacyQuadletSources returns the Quadlet source paths a gare that predates native unit
// synthesis left for an application.
func LegacyQuadletSources(name string) []string {
	dir := filepath.Join(xdg.ConfigHome(), "containers", "systemd")
	return []string{
		filepath.Join(dir, name+legacyKubeExtension),
		filepath.Join(dir, name+legacyContainerExtension),
	}
}

// LegacyQuadletSourcesExist reports whether a previous gare left Quadlet sources for an application.
func LegacyQuadletSourcesExist(name string) (bool, error) {
	for _, path := range LegacyQuadletSources(name) {
		if _, err := os.Stat(path); err == nil {
			return true, nil
		} else if !os.IsNotExist(err) {
			return false, err
		}
	}
	return false, nil
}

// RemoveLegacyQuadletSources deletes the Quadlet sources a previous gare wrote, so a workload gare
// now supervises natively cannot be resurrected by a stale source once its generated unit is gone.
func RemoveLegacyQuadletSources(name string) error {
	for _, path := range LegacyQuadletSources(name) {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}
