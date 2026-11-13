package systemd

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
{{range .Networks}}ExecStartPre={{$.PodmanPath}} network create --ignore {{.}}
{{end}}ExecStart={{.PodmanPath}} compose -f {{.ComposeFile}}{{with .OverrideFile}} -f {{.}}{{end}} -p {{.ProjectName}} up -d
ExecStop=-{{.PodmanPath}} compose -f {{.ComposeFile}}{{with .OverrideFile}} -f {{.}}{{end}} -p {{.ProjectName}} down
ExecStopPost=-{{.PodmanPath}} compose -f {{.ComposeFile}}{{with .OverrideFile}} -f {{.}}{{end}} -p {{.ProjectName}} down
SyslogIdentifier=%N

[Install]
WantedBy=default.target
`

// ComposeUnitData holds template parameters for synthesizing a compose systemd user unit. The
// optional OverrideFile is gare's own compose file layered over the repository's, and Networks are
// the external networks the stack joins, which the unit creates before every start.
type ComposeUnitData struct {
	Name         string
	Description  string
	PodmanPath   string
	RepoDir      string
	ComposeFile  string
	OverrideFile string
	ProjectName  string
	EnvFile      string
	Networks     []string
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
// External networks are created with `network create --ignore` in a plain ExecStartPre, so the unit
// recreates a missing one after a reboot or a `podman system reset` without a shell, and a failed
// creation fails the start instead of letting the provider fail later with a vaguer error.
func GenerateComposeUnit(data ComposeUnitData) (string, error) {
	if data.OverrideFile != "" {
		if err := validateUnitPath("compose override", data.OverrideFile); err != nil {
			return "", err
		}
	}
	if data.PodmanPath == "" {
		data.PodmanPath = ResolvePodmanPath()
	}
	data.Description = ComposeUnitDescription(data.Name)
	return renderUnit("compose-unit", composeUnitTemplate, data)
}

// WriteComposeUnit generates and writes a compose systemd user unit file atomically.
func WriteComposeUnit(data ComposeUnitData) error {
	content, err := GenerateComposeUnit(data)
	if err != nil {
		return err
	}
	return writeUnitFile(data.Name, content)
}
