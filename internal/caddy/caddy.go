package caddy

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/FacileStudio/gare/internal/atomicfile"
)

const snippetTemplate = `{{.Address}} {
		reverse_proxy localhost:{{.Port}}
	}
	`

const staticSnippetTemplate = `{{.Address}} {
		root * "{{.RootDir}}"
		try_files {path} /index.html
		file_server
	}`

// DefaultConfDir defines the default filesystem path for Caddy drop-in configuration snippets.
const DefaultConfDir = "/etc/caddy/conf.d"

// DefaultCaddyfile is the root Caddyfile that imports gare's drop-in configuration snippets.
const DefaultCaddyfile = `/etc/caddy/Caddyfile`

const defaultCaddyfileContent = `import /etc/caddy/conf.d/*.caddy
`

// SnippetData holds the template parameters for generating a Caddy reverse proxy snippet.
type SnippetData struct {
	Address string
	Port    int
}

// StaticSnippetData holds the template parameters for generating a Caddy static file server snippet.
type StaticSnippetData struct {
	Address string
	Port    int
	RootDir string
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

// GenerateStaticSnippet renders a Caddy static file server configuration snippet.
func GenerateStaticSnippet(domains []string, port int, rootDir string) (string, error) {
	var address string
	if len(domains) > 0 && port > 0 {
		address = fmt.Sprintf("%s, :%d", strings.Join(domains, ", "), port)
	} else if len(domains) > 0 {
		address = strings.Join(domains, ", ")
	} else if port > 0 {
		address = fmt.Sprintf(":%d", port)
	} else {
		return "", errors.New("at least one of domain or port must be specified")
	}
	tmpl, err := template.New("caddy-static").Parse(staticSnippetTemplate)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, StaticSnippetData{Address: address, Port: port, RootDir: rootDir}); err != nil {
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

// WriteStaticSnippet generates and writes a static file server configuration snippet atomically to disk.
func WriteStaticSnippet(confDir, name string, domains []string, port int, rootDir string) error {
	if confDir == "" {
		confDir = ResolveConfDir()
	}
	if err := EnsureCaddyfilePaths(ResolveCaddyfilePath(), confDir); err != nil {
		return err
	}
	content, err := GenerateStaticSnippet(domains, port, rootDir)
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

// Reload instructs Caddy to reload its configuration using the CLI or Admin API.
func Reload(ctx context.Context) error {
	if err := EnsureCaddyfile(); err != nil {
		return err
	}
	caddyfile := ResolveCaddyfilePath()
	cmd := exec.CommandContext(ctx, "caddy", "reload", "--config", caddyfile)
	if err := cmd.Run(); err == nil {
		return nil
	}
	return reloadViaAdminAPI(ctx, caddyfile)
}

func reloadViaAdminAPI(ctx context.Context, configPath string) error {
	content, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://127.0.0.1:2019/load", bytes.NewReader(content))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "text/caddyfile")
	resp, reqErr := http.DefaultClient.Do(req)
	if err := CheckReloadError(reqErr); err != nil {
		return err
	}
	if resp == nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("caddy admin api returned %s: %s", resp.Status, string(body))
}
