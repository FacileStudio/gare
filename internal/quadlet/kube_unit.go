package quadlet

import (
	"bytes"
	"fmt"
	"os"
	"text/template"

	"github.com/FacileStudio/gare/internal/atomicfile"
)

const kubeUnitTemplate = `[Unit]
Description=Gare Managed App: {{.Name}}

[Kube]
Yaml="{{.YamlPath}}"
ExitCodePropagation=any

[Service]
Restart=on-failure
RestartSec=5s
TimeoutStartSec=300s
TimeoutStopSec=70s
SyslogIdentifier=%N

[Install]
WantedBy=default.target
`

// KubeUnitData holds template parameters for a Quadlet kube source file.
type KubeUnitData struct {
	Name     string
	YamlPath string
}

// GenerateKubeUnit renders the Quadlet source file supervising a pod manifest.
// ExitCodePropagation=any keeps Restart=on-failure meaningful: quadlet's default (none)
// exits the service zero even when a container failed, so nothing would restart it.
func GenerateKubeUnit(data KubeUnitData) (string, error) {
	if err := validateSourcePath("manifest", data.YamlPath); err != nil {
		return "", err
	}
	tmpl, err := template.New("kube-unit").Parse(kubeUnitTemplate)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// WriteKubeUnit generates and writes the Quadlet source file for an application.
func WriteKubeUnit(name, yamlPath string) error {
	content, err := GenerateKubeUnit(KubeUnitData{Name: name, YamlPath: yamlPath})
	if err != nil {
		return err
	}
	dir := Dir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create quadlet directory %s: %w", dir, err)
	}
	sourcePath := KubePath(name)
	if err := atomicfile.WriteFile(sourcePath, []byte(content), 0644); err != nil {
		return err
	}
	return retireSiblingSources(name, sourcePath)
}
