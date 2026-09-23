package systemd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"

	"github.com/FacileStudio/gare/internal/atomicfile"
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

// GareUnitDescriptions returns every Description= value gare's workload templates write for an
// application. A unit file carrying one of them is one gare wrote itself, so a deploy may replace
// it; a unit carrying none belongs to the operator and must be left alone.
func GareUnitDescriptions(name string) []string {
	return []string{
		KubeUnitDescription(name),
		StaticUnitDescription(name),
		ComposeUnitDescription(name),
	}
}

// GetUnitPath returns the absolute path for an application unit file in the user unit directory.
func GetUnitPath(name string) string {
	return filepath.Join(DefaultUserUnitDir(), name+".service")
}

// renderUnit executes a workload unit template against its data, the single place every workload
// type turns its parameters into unit text.
func renderUnit(name, tmpl string, data any) (string, error) {
	parsed, err := template.New(name).Parse(tmpl)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := parsed.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// writeUnitFile writes a rendered unit atomically into the user unit directory, creating that
// directory first because gare may be the first thing to do so on a fresh account.
func writeUnitFile(name, content string) error {
	unitPath := GetUnitPath(name)
	dir := filepath.Dir(unitPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create unit directory %s: %w", dir, err)
	}
	return atomicfile.WriteFile(unitPath, []byte(content), 0644)
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
