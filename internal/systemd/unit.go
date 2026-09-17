package systemd

import (
	"bytes"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"text/template"

	"github.com/FacileStudio/gare/internal/atomicfile"
)

const unitTemplate = `[Unit]
Description=Gare Managed App: {{.Name}}

[Service]
Environment=PODMAN_SYSTEMD_UNIT=%n
Type=exec
KillMode=mixed
Delegate=yes
Restart=on-failure
RestartSec=5s
TimeoutStopSec=70s
ExecStart={{.PodmanPath}} kube play --replace -w {{.ManifestPath}}
ExecStopPost=-{{.PodmanPath}} kube down {{.ManifestPath}}
SyslogIdentifier=%N

[Install]
WantedBy=default.target
`

// UnitData holds template parameters for synthesizing a systemd user unit.
type UnitData struct {
	Name         string
	PodmanPath   string
	ManifestPath string
}

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

// GenerateUnit renders a systemd service unit file for the given application and manifest path.
func GenerateUnit(name, manifestPath string) (string, error) {
	tmpl, err := template.New("unit").Parse(unitTemplate)
	if err != nil {
		return "", err
	}

	data := UnitData{
		Name:         name,
		PodmanPath:   ResolvePodmanPath(),
		ManifestPath: manifestPath,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// WriteUnit generates and writes a systemd service unit file atomically to disk.
func WriteUnit(name, manifestPath string) error {
	content, err := GenerateUnit(name, manifestPath)
	if err != nil {
		return err
	}

	unitPath := GetUnitPath(name)
	return atomicfile.WriteFile(unitPath, []byte(content), 0644)
}

// RemoveUnit deletes the systemd service unit file for the given application.
func RemoveUnit(name string) error {
	unitPath := GetUnitPath(name)
	if err := os.Remove(unitPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
