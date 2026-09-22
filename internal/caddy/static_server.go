package caddy

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/FacileStudio/gare/internal/atomicfile"
)

const staticServerConfigTemplate = `:{{.Port}} {
	root * "{{.RootMount}}"
	try_files {path} /index.html
	file_server
}
`

// StaticServerData holds template parameters for a static workload's Caddyfile.
type StaticServerData struct {
	Port      int
	RootMount string
}

// GenerateStaticServerConfig renders the Caddyfile a static workload's container serves.
func GenerateStaticServerConfig(rootMount string, port int) (string, error) {
	if port <= 0 || port > 65535 {
		return "", fmt.Errorf("static server config requires a port between 1 and 65535, got %d", port)
	}
	tmpl, err := template.New("caddy-static-server").Parse(staticServerConfigTemplate)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, StaticServerData{Port: port, RootMount: rootMount}); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// WriteStaticServerConfig generates and writes a static workload's Caddyfile atomically.
func WriteStaticServerConfig(path, rootMount string, port int) error {
	content, err := GenerateStaticServerConfig(rootMount, port)
	if err != nil {
		return err
	}
	return atomicfile.WriteFile(path, []byte(content), 0644)
}
