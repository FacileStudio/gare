package systemd

const kubeUnitTemplate = `[Unit]
Description={{.Description}}
After=podman-user-wait-network-online.service
Wants=podman-user-wait-network-online.service
RequiresMountsFor=%t/containers

[Service]
Type=notify
NotifyAccess=all
Environment=PODMAN_SYSTEMD_UNIT=%n
KillMode=mixed
ExecStart={{.PodmanPath}} kube play --replace --service-container=true --service-exit-code-propagation=any {{.YamlPath}}
ExecStopPost={{.PodmanPath}} kube down {{.YamlPath}}
Restart=on-failure
RestartSec=5s
TimeoutStartSec=300s
TimeoutStopSec=70s
SyslogIdentifier=%N

[Install]
WantedBy=default.target
`

// KubeUnitData holds template parameters for a pod manifest's systemd unit.
type KubeUnitData struct {
	Name        string
	Description string
	YamlPath    string
	PodmanPath  string
}

// KubeUnitDescription returns the Description= value of a container workload's unit, the marker gare
// recognises a unit it wrote itself by.
func KubeUnitDescription(name string) string {
	return "Gare Managed App: " + name
}

// GenerateKubeUnit renders the systemd unit supervising a pod manifest.
// --service-exit-code-propagation=any is what keeps Restart=on-failure real: podman otherwise
// exits the service zero even when a container failed, so nothing would ever restart it.
func GenerateKubeUnit(data KubeUnitData) (string, error) {
	if err := validateUnitPath("manifest", data.YamlPath); err != nil {
		return "", err
	}
	if data.PodmanPath == "" {
		data.PodmanPath = ResolvePodmanPath()
	}
	data.Description = KubeUnitDescription(data.Name)
	return renderUnit("kube-unit", kubeUnitTemplate, data)
}

// WriteKubeUnit generates and writes the systemd unit supervising an application's pod manifest.
func WriteKubeUnit(name, yamlPath string) error {
	content, err := GenerateKubeUnit(KubeUnitData{Name: name, YamlPath: yamlPath})
	if err != nil {
		return err
	}
	return writeUnitFile(name, content)
}
