package systemd

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/FacileStudio/gare/internal/xdg"
)

// ResolvePodmanPath locates the podman executable in PATH or defaults to /usr/bin/podman.
func ResolvePodmanPath() string {
	path, err := exec.LookPath("podman")
	if err != nil || path == "" {
		return "/usr/bin/podman"
	}
	return path
}

// DefaultUserUnitDir returns the systemd user unit directory for the current user.
func DefaultUserUnitDir() string {
	return filepath.Join(xdg.ConfigHome(), "systemd", "user")
}

// GetUnitPath returns the absolute path for an application unit file in the user unit directory.
func GetUnitPath(name string) string {
	return filepath.Join(DefaultUserUnitDir(), name+".service")
}

// RemoveUnit deletes the systemd service unit file for the given application, together with the
// enable link systemd created for it, which would otherwise outlive the unit as a dangling target.
func RemoveUnit(name string) error {
	if err := os.Remove(GetUnitPath(name)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return removeEnableLink(name)
}

// removeEnableLink deletes the default.target enable link of a unit, the one location every
// gare-written unit installs itself into.
func removeEnableLink(name string) error {
	linkPath := filepath.Join(DefaultUserUnitDir(), "default.target.wants", name+".service")
	if err := os.Remove(linkPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
