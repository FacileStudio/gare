package caddy

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"

	"github.com/FacileStudio/gare/internal/atomicfile"
)

const snippetTemplate = `{{.Domain}} {
	reverse_proxy localhost:{{.Port}}
}
`

const staticSnippetTemplate = `{{.Domain}} {
	root * "{{.RootDir}}"
	file_server {
		try_files {path} /index.html
	}
}
`

// DefaultConfDir defines the default filesystem path for Caddy drop-in configuration snippets.
const DefaultConfDir = "/etc/caddy/conf.d"

// DefaultCaddyfile is the root Caddyfile that imports gare's drop-in configuration snippets.
const DefaultCaddyfile = `/etc/caddy/Caddyfile`

const defaultCaddyfileContent = `import /etc/caddy/conf.d/*.caddy
`

// SnippetData holds the template parameters for generating a Caddy reverse proxy snippet.
type SnippetData struct {
	Domain string
	Port   int
}

// StaticSnippetData holds the template parameters for generating a Caddy static file server snippet.
type StaticSnippetData struct {
	Domain  string
	RootDir string
}

// GenerateSnippet renders a Caddy reverse proxy configuration snippet for the given domain and port.
func GenerateSnippet(domain string, port int) (string, error) {
	tmpl, err := template.New("caddy").Parse(snippetTemplate)
	if err != nil {
		return "", err
	}

	data := SnippetData{
		Domain: domain,
		Port:   port,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// GenerateStaticSnippet renders a Caddy static file server configuration snippet.
func GenerateStaticSnippet(domain, rootDir string) (string, error) {
	tmpl, err := template.New("caddy-static").Parse(staticSnippetTemplate)
	if err != nil {
		return "", err
	}

	data := StaticSnippetData{
		Domain:  domain,
		RootDir: rootDir,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// GetSnippetPath returns the absolute path for an application snippet file in the given directory.
func GetSnippetPath(confDir, name string) string {
	if confDir == "" {
		confDir = DefaultConfDir
	}
	return filepath.Join(confDir, fmt.Sprintf("%s.caddy", name))
}

// WriteSnippet generates and writes a Caddy configuration snippet atomically to disk.
func WriteSnippet(confDir, name, domain string, port int) error {
	content, err := GenerateSnippet(domain, port)
	if err != nil {
		return err
	}

	path := GetSnippetPath(confDir, name)
	return atomicfile.WriteFile(path, []byte(content), 0644)
}

// WriteStaticSnippet generates and writes a static file server configuration snippet atomically to disk.
func WriteStaticSnippet(confDir, name, domain, rootDir string) error {
	content, err := GenerateStaticSnippet(domain, rootDir)
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

// Reload instructs Caddy to reload its configuration, falling back to systemctl if needed.
func Reload(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "caddy", "reload", "--config", DefaultCaddyfile)
	if err := cmd.Run(); err == nil {
		return nil
	}
	fallbackCmd := exec.CommandContext(ctx, "systemctl", "reload", "caddy")
	return fallbackCmd.Run()
}
