package storage

import (
	"bytes"
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
    - containerPort: {{.Port}}
      hostPort: {{.Port}}
`))

// GetRepoDir returns the repository directory path within an app directory.
func GetRepoDir(appDir string) string {
	return filepath.Join(appDir, "repo")
}

// GetManifestPath returns the manifest.yaml path within an app directory.
func GetManifestPath(appDir string) string {
	return filepath.Join(appDir, "manifest.yaml")
}

// GenerateDefaultManifest generates a default Kubernetes Pod manifest.
func GenerateDefaultManifest(name string, port int, destPath string) error {
	targetFile := destPath
	if fi, err := os.Stat(destPath); (err == nil && fi.IsDir()) || filepath.Ext(destPath) == "" {
		targetFile = filepath.Join(destPath, "manifest.yaml")
	}
	var buf bytes.Buffer
	data := struct {
		Name string
		Port int
	}{
		Name: name,
		Port: port,
	}
	if err := defaultManifestTemplate.Execute(&buf, data); err != nil {
		return err
	}
	return atomicfile.WriteFile(targetFile, buf.Bytes(), 0644)
}
