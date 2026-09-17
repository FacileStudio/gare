package storage

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/FacileStudio/gare/internal/atomicfile"
)

var defaultManifestTemplate = template.Must(template.New("manifest").Parse(`apiVersion: v1
kind: Pod
metadata:
  name: {{.Name}}
  labels:
    app: {{.Name}}
spec:
  containers:
  - name: {{.Name}}
    image: localhost/{{.Name}}:latest
    ports:
    - containerPort: {{.ContainerPort}}
      hostPort: {{.HostPort}}
`))

// GetRepoDir returns the repository directory path within an app directory.
func GetRepoDir(appDir string) string {
	return filepath.Join(appDir, "repo")
}

// GetManifestPath returns the manifest.yaml path within an app directory.
func GetManifestPath(appDir string) string {
	return filepath.Join(appDir, "manifest.yaml")
}

// GenerateDefaultManifest generates a default Kubernetes Pod manifest with separate container and host ports.
func GenerateDefaultManifest(name string, containerPort int, hostPort int, destPath string) error {
	if hostPort <= 0 || hostPort > 65535 {
		return fmt.Errorf("hostPort must be between 1 and 65535, got %d", hostPort)
	}
	if containerPort <= 0 {
		containerPort = hostPort
	}
	if containerPort > 65535 {
		return fmt.Errorf("containerPort must be between 1 and 65535, got %d", containerPort)
	}
	targetFile := destPath
	if fi, err := os.Stat(destPath); (err == nil && fi.IsDir()) || filepath.Ext(destPath) == "" {
		targetFile = filepath.Join(destPath, "manifest.yaml")
	}
	var buf bytes.Buffer
	data := struct {
		Name          string
		ContainerPort int
		HostPort      int
	}{
		Name:          name,
		ContainerPort: containerPort,
		HostPort:      hostPort,
	}
	if err := defaultManifestTemplate.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to execute manifest template: %w", err)
	}
	return atomicfile.WriteFile(targetFile, buf.Bytes(), 0644)
}
