package systemd

import (
	"bytes"
	"text/template"

	"github.com/FacileStudio/gare/internal/atomicfile"
)

const composeUnitTemplate = `[Unit]
Description=Gare Managed Compose App: {{.Name}}
After=network-online.target podman.socket
Wants=network-online.target
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
	PodmanPath  string
	RepoDir     string
	ComposeFile string
	ProjectName string
	EnvFile     string
}

// GenerateComposeUnit renders a systemd service unit for a supervised compose workload.
func GenerateComposeUnit(data ComposeUnitData) (string, error) {
	tmpl, err := template.New("compose-unit").Parse(composeUnitTemplate)
	if err != nil {
		return "", err
	}
	if data.PodmanPath == "" {
		data.PodmanPath = ResolvePodmanPath()
	}

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
