package manifest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
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
    imagePullPolicy: Never
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

type pullPolicyContainer struct {
	Name            string `yaml:"name"`
	Image           string `yaml:"image"`
	ImagePullPolicy string `yaml:"imagePullPolicy"`
}

type pullPolicyDoc struct {
	Spec struct {
		Containers []pullPolicyContainer `yaml:"containers"`
	} `yaml:"spec"`
}

func writePullPolicyManifest(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "manifest.yaml")
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func readPullPolicyManifest(t *testing.T, path string) pullPolicyDoc {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc pullPolicyDoc
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("failed to parse manifest: %v", err)
	}
	return doc
}

func TestEnsureLocalImagePullPolicy(t *testing.T) {
	body := "apiVersion: v1\nkind: Pod\nspec:\n  containers:\n" +
		"  - name: api\n    image: localhost/stack:latest\n" +
		"  - name: db\n    image: docker.io/library/postgres:16-alpine\n" +
		"  - name: pinned\n    image: localhost/other:latest\n    imagePullPolicy: Always\n"
	path := writePullPolicyManifest(t, body)
	if err := EnsureLocalImagePullPolicy(path); err != nil {
		t.Fatalf("EnsureLocalImagePullPolicy failed: %v", err)
	}
	doc := readPullPolicyManifest(t, path)
	want := map[string]string{"api": "Never", "db": "", "pinned": "Always"}
	for _, container := range doc.Spec.Containers {
		if container.ImagePullPolicy != want[container.Name] {
			t.Errorf("container %s: got policy %q, want %q",
				container.Name, container.ImagePullPolicy, want[container.Name])
		}
	}
}

func TestEnsureLocalImagePullPolicyIsIdempotent(t *testing.T) {
	path := writePullPolicyManifest(t, "apiVersion: v1\nkind: Pod\nspec:\n  containers:\n"+
		"  - name: api\n    image: localhost/stack:latest\n")
	for range 2 {
		if err := EnsureLocalImagePullPolicy(path); err != nil {
			t.Fatalf("EnsureLocalImagePullPolicy failed: %v", err)
		}
	}
	if got := strings.Count(string(mustRead(t, path)), "imagePullPolicy"); got != 1 {
		t.Errorf("expected exactly one imagePullPolicy key, got %d", got)
	}
}

func TestEnsureLocalImagePullPolicyMissingFile(t *testing.T) {
	if err := EnsureLocalImagePullPolicy(filepath.Join(t.TempDir(), "absent.yaml")); err == nil {
		t.Error("expected an error for a missing manifest")
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
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
