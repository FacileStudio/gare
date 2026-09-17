package systemd

import (
	"bytes"
	"os/exec"
	"text/template"

	"github.com/FacileStudio/gare/internal/atomicfile"
)

const staticUnitTemplate = `[Unit]
Description=Gare Managed Static App: {{.Name}}
After=network-online.target
Wants=network-online.target

[Service]
Type=exec
Restart=on-failure
RestartSec=5s
EnvironmentFile=-%h/.local/share/gare/apps/{{.Name}}/env
ExecStart={{.CaddyPath}} file-server --listen :{{.Port}} --root {{.RootDir}}
SyslogIdentifier=%N

[Install]
WantedBy=default.target
`

// StaticUnitData holds template parameters for synthesizing a static systemd user unit.
type StaticUnitData struct {
	Name      string
	CaddyPath string
	Port      int
	RootDir   string
}

// ResolveCaddyPath locates the caddy executable in PATH or defaults to /usr/bin/caddy.
func ResolveCaddyPath() string {
	path, err := exec.LookPath("caddy")
	if err != nil || path == "" {
		return "/usr/bin/caddy"
	}
	return path
}

// GenerateStaticUnit renders a systemd service unit file for a static application.
func GenerateStaticUnit(name string, port int, rootDir string) (string, error) {
	tmpl, err := template.New("static-unit").Parse(staticUnitTemplate)
	if err != nil {
		return "", err
	}

	data := StaticUnitData{
		Name:      name,
		CaddyPath: ResolveCaddyPath(),
		Port:      port,
		RootDir:   rootDir,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// WriteStaticUnit generates and writes a static systemd service unit file atomically to disk.
func WriteStaticUnit(name string, port int, rootDir string) error {
	content, err := GenerateStaticUnit(name, port, rootDir)
	if err != nil {
		return err
	}

	unitPath := GetUnitPath(name)
	return atomicfile.WriteFile(unitPath, []byte(content), 0644)
}
