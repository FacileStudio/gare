package ingress

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/FacileStudio/gare/apps/api/internal/atomicfile"
)

const snippetTemplate = `{{.Address}} {
	reverse_proxy localhost:{{.Port}}
}
`

// DefaultConfDir defines the default filesystem path for Caddy drop-in configuration snippets.
const DefaultConfDir = "/etc/caddy/conf.d"

// DefaultCaddyfile is the root Caddyfile that imports gare's drop-in configuration snippets.
const DefaultCaddyfile = `/etc/caddy/Caddyfile`

// SnippetData holds the template parameters for generating a Caddy reverse proxy snippet.
type SnippetData struct {
	Address string
	Port    int
}

// GenerateSnippet renders a Caddy reverse proxy configuration snippet for the given domains and port.
func GenerateSnippet(domains []string, port int) (string, error) {
	if len(domains) == 0 {
		return "", errors.New("at least one domain must be specified")
	}
	address := strings.Join(domains, ", ")
	tmpl, err := template.New("caddy").Parse(snippetTemplate)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, SnippetData{Address: address, Port: port}); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// GetSnippetPath returns the absolute path for an application snippet file in the given directory.
func GetSnippetPath(confDir, name string) string {
	if confDir == "" {
		confDir = ResolveConfDir()
	}
	return filepath.Join(confDir, fmt.Sprintf("%s.caddy", name))
}

// WriteSnippet generates and writes a Caddy configuration snippet atomically to disk.
func WriteSnippet(confDir, name string, domains []string, port int) error {
	if confDir == "" {
		confDir = ResolveConfDir()
	}
	if err := EnsureCaddyfilePaths(ResolveCaddyfilePath(), confDir); err != nil {
		return err
	}
	content, err := GenerateSnippet(domains, port)
	if err != nil {
		return err
	}
	path := GetSnippetPath(confDir, name)
	return atomicfile.WriteFile(path, []byte(content), 0644)
}

// RemoveSnippet deletes the Caddy configuration snippet file for the given application.
func RemoveSnippet(confDir, name string) error {
	path := GetSnippetPath(confDir, name)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
