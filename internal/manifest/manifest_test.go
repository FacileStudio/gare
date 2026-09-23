package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

const expectedManifest = `apiVersion: v1
kind: Pod
metadata:
  name: webapp
  labels:
    app: webapp
spec:
  containers:
  - name: webapp
    image: localhost/webapp:latest
    ports:
    - containerPort: 3011
      hostPort: 8080
`

func TestGenerate(t *testing.T) {
	manifestFile := filepath.Join(t.TempDir(), "manifest.yaml")
	if err := Generate("webapp", 3011, 8080, manifestFile); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	content, err := os.ReadFile(manifestFile)
	if err != nil || string(content) != expectedManifest {
		t.Fatalf("manifest content mismatch: %v", err)
	}
}

func TestGenerateIntoDirectory(t *testing.T) {
	appDir := filepath.Join(t.TempDir(), "appwithdir")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := Generate("webapp2", 3011, 9000, appDir); err != nil {
		t.Fatalf("Generate with dir failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(appDir, "manifest.yaml")); err != nil {
		t.Fatalf("expected manifest.yaml to be created: %v", err)
	}
}
