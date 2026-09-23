package systemd

import (
	"fmt"
)

const (
	defaultStaticImage  = "docker.io/library/caddy:2-alpine"
	staticContainerPort = 80
	staticRootMount     = "/srv"
	staticConfigMount   = "/etc/caddy/Caddyfile"
)

// containerUnitTemplate renders a static site's unit. The command after the image spells the caddy
// binary out in full because the official image declares no Entrypoint: its Cmd carries the whole
// invocation, so appending arguments replaces it and the runtime then tries to exec the first
// argument as a binary of its own.
const containerUnitTemplate = `[Unit]
Description={{.Description}}
After=podman-user-wait-network-online.service
Wants=podman-user-wait-network-online.service
RequiresMountsFor=%t/containers
RequiresMountsFor={{.RootDir}}
RequiresMountsFor={{.ConfigFile}}

[Service]
Type=notify
NotifyAccess=all
Environment=PODMAN_SYSTEMD_UNIT=%n
KillMode=mixed
Delegate=yes
ExecStart={{.PodmanPath}} run --name {{.Name}} --cidfile=%t/%N.cid --replace --rm --cgroups=split --sdnotify=conmon -d -v {{.RootDir}}:{{.RootMount}}:ro,Z -v {{.ConfigFile}}:{{.ConfigMount}}:ro,Z --publish {{.Port}}:{{.ContainerPort}} --env-file {{.EnvFile}} {{.Image}} /usr/bin/caddy run --config {{.ConfigMount}} --adapter caddyfile
ExecStop={{.PodmanPath}} rm -v -f -i --cidfile=%t/%N.cid
ExecStopPost=-{{.PodmanPath}} rm -v -f -i --cidfile=%t/%N.cid
Restart=on-failure
RestartSec=5s
TimeoutStartSec=300s
SyslogIdentifier=%N

[Install]
WantedBy=default.target
`

// ContainerUnitData holds template parameters for a static site's systemd unit.
type ContainerUnitData struct {
	Name          string
	Description   string
	Image         string
	Port          int
	ContainerPort int
	RootDir       string
	RootMount     string
	ConfigFile    string
	ConfigMount   string
	EnvFile       string
	PodmanPath    string
}

// StaticUnitDescription returns the Description= value of a static workload's unit, the marker gare
// recognises a unit it wrote itself by.
func StaticUnitDescription(name string) string {
	return "Gare Managed Static App: " + name
}

// StaticSiteUnit builds the unit data serving a static directory with the bundled Caddy image.
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

// GenerateContainerUnit renders the systemd unit running a static site's container.
// The unit names the server binary itself rather than leaning on the image, and teardown goes
// through the cidfile so it removes the exact container the unit started.
func GenerateContainerUnit(data ContainerUnitData) (string, error) {
	if err := validateUnitPath("static root", data.RootDir); err != nil {
		return "", err
	}
	if err := validateUnitPath("env file", data.EnvFile); err != nil {
		return "", err
	}
	if err := validateUnitPath("caddyfile", data.ConfigFile); err != nil {
		return "", err
	}
	if data.Port <= 0 || data.Port > 65535 {
		return "", fmt.Errorf("static workloads require a host port between 1 and 65535, got %d", data.Port)
	}
	if data.ContainerPort <= 0 || data.ContainerPort > 65535 {
		return "", fmt.Errorf("static workloads require a container port between 1 and 65535, got %d", data.ContainerPort)
	}
	if data.PodmanPath == "" {
		data.PodmanPath = ResolvePodmanPath()
	}
	data.Description = StaticUnitDescription(data.Name)
	return renderUnit("container-unit", containerUnitTemplate, data)
}

// WriteContainerUnit generates and writes the systemd unit running an application's static site.
func WriteContainerUnit(data ContainerUnitData) error {
	content, err := GenerateContainerUnit(data)
	if err != nil {
		return err
	}
	return writeUnitFile(data.Name, content)
}
