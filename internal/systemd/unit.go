package systemd

import (
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
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
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		if u, err := user.Current(); err == nil && u.HomeDir != "" {
			home = u.HomeDir
		} else {
			home = os.Getenv("HOME")
		}
	}
	return filepath.Join(home, ".config", "systemd", "user")
}

// GetUnitPath returns the absolute path for an application unit file in the user unit directory.
func GetUnitPath(name string) string {
	return filepath.Join(DefaultUserUnitDir(), name+".service")
}

// RemoveUnit deletes the systemd service unit file for the given application.
func RemoveUnit(name string) error {
	unitPath := GetUnitPath(name)
	if err := os.Remove(unitPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
