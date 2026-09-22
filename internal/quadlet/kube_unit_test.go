package quadlet

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateKubeUnit(t *testing.T) {
	content, err := GenerateKubeUnit(KubeUnitData{
		Name:     "my-app",
		YamlPath: "/home/user/.local/share/gare/apps/my-app/manifest.yaml",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedSnippets := []string{
		"Description=Gare Managed App: my-app",
		"[Kube]",
		`Yaml="/home/user/.local/share/gare/apps/my-app/manifest.yaml"`,
		"ExitCodePropagation=any",
		"Restart=on-failure",
		"RestartSec=5s",
		"TimeoutStopSec=70s",
		"[Install]",
		"WantedBy=default.target",
	}
	for _, snippet := range expectedSnippets {
		if !strings.Contains(content, snippet) {
			t.Errorf("expected source to contain %q, got:\n%s", snippet, content)
		}
	}
}

func TestGenerateKubeUnitLeavesCgroupSetupToQuadlet(t *testing.T) {
	content, err := GenerateKubeUnit(KubeUnitData{Name: "my-app", YamlPath: "/srv/manifest.yaml"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(content, "Delegate=") {
		t.Errorf("quadlet owns cgroup setup; gare must not set Delegate, got:\n%s", content)
	}
}

func TestGenerateKubeUnitLeavesExecutionToQuadlet(t *testing.T) {
	content, err := GenerateKubeUnit(KubeUnitData{Name: "my-app", YamlPath: "/srv/manifest.yaml"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, forbidden := range []string{"ExecStart=", "ExecStop", "kube play", "podman"} {
		if strings.Contains(content, forbidden) {
			t.Errorf("quadlet must own %q, got:\n%s", forbidden, content)
		}
	}
}

func TestGenerateKubeUnitRejectsRelativePath(t *testing.T) {
	if _, err := GenerateKubeUnit(KubeUnitData{Name: "my-app", YamlPath: "manifest.yaml"}); err == nil {
		t.Error("expected an error for a relative manifest path")
	}
}

func TestGenerateKubeUnitRejectsUnparseablePath(t *testing.T) {
	invalid := []string{
		"",
		"   ",
		" /srv/manifest.yaml",
		"/srv/manifest.yaml ",
		"/srv/\"manifest\".yaml",
		"/srv/manifest\n.yaml",
	}
	for _, path := range invalid {
		if _, err := GenerateKubeUnit(KubeUnitData{Name: "my-app", YamlPath: path}); err == nil {
			t.Errorf("expected an error for manifest path %q", path)
		}
	}
}

func TestWriteKubeUnit(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	manifestPath := filepath.Join(configHome, "apps", "site", "manifest.yaml")
	if err := WriteKubeUnit("site", manifestPath); err != nil {
		t.Fatalf("WriteKubeUnit failed: %v", err)
	}

	data, err := os.ReadFile(KubePath("site"))
	if err != nil {
		t.Fatalf("quadlet source was not written: %v", err)
	}
	if !strings.Contains(string(data), "Yaml=\""+manifestPath+"\"") {
		t.Errorf("source is missing the manifest path:\n%s", string(data))
	}
}

func TestWriteKubeUnitRejectsInvalidPath(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := WriteKubeUnit("site", "manifest.yaml"); err == nil {
		t.Error("expected an error for a relative manifest path")
	}
}

func TestGeneratedSourceIsAcceptedByQuadlet(t *testing.T) {
	unitDir := t.TempDir()
	manifestPath := filepath.Join(unitDir, "manifest.yaml")
	manifest := "apiVersion: v1\nkind: Pod\nmetadata:\n  name: site\n"
	if err := os.WriteFile(manifestPath, []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}

	source, err := GenerateKubeUnit(KubeUnitData{Name: "site", YamlPath: manifestPath})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(unitDir, "site.kube"), []byte(source), 0644); err != nil {
		t.Fatal(err)
	}

	generated := runQuadletDryRun(t, unitDir)
	for _, snippet := range []string{"---site.service---", "--service-container=true", "ExitCodePropagation=any", "WantedBy=default.target"} {
		if !strings.Contains(generated, snippet) {
			t.Errorf("expected the generated unit to contain %q, got:\n%s", snippet, generated)
		}
	}
}
