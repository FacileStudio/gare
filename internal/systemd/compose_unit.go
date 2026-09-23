package systemd

import (
	"bytes"
	"text/template"

	"github.com/FacileStudio/gare/internal/atomicfile"
)

const composeUnitTemplate = `[Unit]
Description={{.Description}}
After=podman-user-wait-network-online.service podman.socket
Wants=podman-user-wait-network-online.service
Requires=podman.socket

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory={{.RepoDir}}
Environment=PODMAN_SYSTEMD_UNIT=%n
EnvironmentFile=-{{.EnvFile}}
KillMode=mixed
Delegate=yes
Restart=on-failure
RestartSec=5s
TimeoutStartSec=300s
TimeoutStopSec=70s
ExecStart={{.PodmanPath}} compose -f {{.ComposeFile}} -p {{.ProjectName}} up -d
ExecStop=-{{.PodmanPath}} compose -f {{.ComposeFile}} -p {{.ProjectName}} down
ExecStopPost=-{{.PodmanPath}} compose -f {{.ComposeFile}} -p {{.ProjectName}} down
SyslogIdentifier=%N

[Install]
WantedBy=default.target
`

// ComposeUnitData holds template parameters for synthesizing a compose systemd user unit.
type ComposeUnitData struct {
	Name        string
	Description string
	PodmanPath  string
	RepoDir     string
	ComposeFile string
	ProjectName string
	EnvFile     string
}

// ComposeUnitDescription returns the Description= value of a compose workload's unit, the marker gare
// recognises a unit it wrote itself by.
func ComposeUnitDescription(name string) string {
	return "Gare Managed Compose App: " + name
}

// GenerateComposeUnit renders a systemd service unit for a supervised compose workload.
// The network ordering goes through podman-user-wait-network-online.service rather than
// network-online.target, because the user manager has no network-online.target: systemd drops
// ordering on an unknown unit silently, leaving the provider to run before the network is up.
func GenerateComposeUnit(data ComposeUnitData) (string, error) {
	tmpl, err := template.New("compose-unit").Parse(composeUnitTemplate)
	if err != nil {
		return "", err
	}
	if data.PodmanPath == "" {
		data.PodmanPath = ResolvePodmanPath()
	}
	data.Description = ComposeUnitDescription(data.Name)

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// WriteComposeUnit generates and writes a compose systemd user unit file atomically.
func WriteComposeUnit(data ComposeUnitData) error {
	content, err := GenerateComposeUnit(data)
	if err != nil {
		return err
	}

	return atomicfile.WriteFile(GetUnitPath(data.Name), []byte(content), 0644)
}
