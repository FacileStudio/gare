package quadlet

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/FacileStudio/gare/internal/atomicfile"
)

const (
	defaultStaticImage  = "docker.io/library/caddy:2-alpine"
	staticContainerPort = 80
	staticRootMount     = "/srv"
	staticConfigMount   = "/etc/caddy/Caddyfile"
)

const containerUnitTemplate = `[Unit]
Description=Gare Managed Static App: {{.Name}}

[Container]
Image={{.Image}}
ContainerName={{.Name}}
Entrypoint=caddy
Exec=run --config {{.ConfigMount}} --adapter caddyfile
PublishPort={{.Port}}:{{.ContainerPort}}
Volume={{.RootDir}}:{{.RootMount}}:ro,Z
Volume={{.ConfigFile}}:{{.ConfigMount}}:ro,Z
EnvironmentFile={{.EnvFile}}

[Service]
Restart=on-failure
RestartSec=5s
TimeoutStartSec=300s
SyslogIdentifier=%N

[Install]
WantedBy=default.target
`

// ContainerUnitData holds template parameters for a Quadlet container source file.
type ContainerUnitData struct {
	Name          string
	Image         string
	Port          int
	ContainerPort int
	RootDir       string
	RootMount     string
	ConfigFile    string
	ConfigMount   string
	EnvFile       string
}

// StaticSiteUnit builds the Quadlet data serving a static directory with the bundled Caddy image.
// The Caddyfile is mounted rather than generated in-container, because caddy file-server cannot
// express the SPA fallback a static site needs.
func StaticSiteUnit(name string, port int, rootDir, envFile, configFile string) ContainerUnitData {
	return ContainerUnitData{
		Name:          name,
		Image:         defaultStaticImage,
		Port:          port,
		ContainerPort: staticContainerPort,
		RootDir:       rootDir,
		RootMount:     staticRootMount,
		ConfigFile:    configFile,
		ConfigMount:   staticConfigMount,
		EnvFile:       envFile,
	}
}

// GenerateContainerUnit renders the Quadlet source file running a container for an application.
func GenerateContainerUnit(data ContainerUnitData) (string, error) {
	if err := validateSourcePath("static root", data.RootDir); err != nil {
		return "", err
	}
	if err := validateSourcePath("env file", data.EnvFile); err != nil {
		return "", err
	}
	if err := validateSourcePath("caddyfile", data.ConfigFile); err != nil {
		return "", err
	}
	if data.Port <= 0 || data.Port > 65535 {
		return "", fmt.Errorf("static workloads require a host port between 1 and 65535, got %d", data.Port)
	}
	if data.ContainerPort <= 0 || data.ContainerPort > 65535 {
		return "", fmt.Errorf("static workloads require a container port between 1 and 65535, got %d", data.ContainerPort)
	}
	tmpl, err := template.New("container-unit").Parse(containerUnitTemplate)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// WriteContainerUnit generates and writes the Quadlet source file for an application container.
func WriteContainerUnit(data ContainerUnitData) error {
	content, err := GenerateContainerUnit(data)
	if err != nil {
		return err
	}
	dir := Dir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create quadlet directory %s: %w", dir, err)
	}
	sourcePath := ContainerPath(data.Name)
	if err := atomicfile.WriteFile(sourcePath, []byte(content), 0644); err != nil {
		return err
	}
	return retireSiblingSources(data.Name, sourcePath)
}

// validateSourcePath rejects paths Quadlet cannot carry into a generated command.
// Whitespace is excluded because Quadlet does not re-quote volume and env-file values.
func validateSourcePath(label, path string) error {
	if path == "" {
		return fmt.Errorf("quadlet sources require a %s path", label)
	}
	if !filepath.IsAbs(path) {
		return fmt.Errorf("quadlet sources require an absolute %s path, got %q", label, path)
	}
	if strings.ContainsAny(path, " \t\n\r\"") {
		return fmt.Errorf("%s path %q contains characters quadlet cannot parse", label, path)
	}
	return nil
}
