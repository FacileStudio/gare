package systemd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateKubeUnit(t *testing.T) {
	content, err := GenerateKubeUnit(KubeUnitData{
		Name:       "my-app",
		YamlPath:   "/home/user/.local/share/gare/apps/my-app/manifest.yaml",
		PodmanPath: "/usr/bin/podman",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedSnippets := []string{
		"Description=Gare Managed App: my-app",
		"ExecStart=/usr/bin/podman kube play --replace --service-container=true --service-exit-code-propagation=any /home/user/.local/share/gare/apps/my-app/manifest.yaml",
		"ExecStopPost=/usr/bin/podman kube down /home/user/.local/share/gare/apps/my-app/manifest.yaml",
		"Type=notify",
		"NotifyAccess=all",
		"Restart=on-failure",
		"RestartSec=5s",
		"TimeoutStopSec=70s",
		"[Install]",
		"WantedBy=default.target",
	}
	for _, snippet := range expectedSnippets {
		if !strings.Contains(content, snippet) {
			t.Errorf("expected unit to contain %q, got:\n%s", snippet, content)
		}
	}
}

func TestGenerateKubeUnitPropagatesContainerExitCode(t *testing.T) {
	content, err := GenerateKubeUnit(KubeUnitData{Name: "my-app", YamlPath: "/srv/manifest.yaml", PodmanPath: "/usr/bin/podman"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(content, "--service-exit-code-propagation=any") {
		t.Errorf("Restart=on-failure stays inert without exit code propagation, got:\n%s", content)
	}
	if !strings.Contains(content, "--service-container=true") {
		t.Errorf("Type=notify never becomes ready without the podman service container, got:\n%s", content)
	}
}

func TestGenerateKubeUnitResolvesPodmanPath(t *testing.T) {
	content, err := GenerateKubeUnit(KubeUnitData{Name: "my-app", YamlPath: "/srv/manifest.yaml"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(content, "ExecStart="+ResolvePodmanPath()+" kube play") {
		t.Errorf("expected the resolved podman path, got:\n%s", content)
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

	data, err := os.ReadFile(GetUnitPath("site"))
	if err != nil {
		t.Fatalf("unit was not written: %v", err)
	}
	if !strings.Contains(string(data), manifestPath) {
		t.Errorf("unit is missing the manifest path:\n%s", string(data))
	}
}

func TestWriteKubeUnitRejectsInvalidPath(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := WriteKubeUnit("site", "manifest.yaml"); err == nil {
		t.Error("expected an error for a relative manifest path")
	}
}

func TestGeneratedKubeUnitIsAcceptedBySystemd(t *testing.T) {
	unitDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", unitDir)

	manifestPath := filepath.Join(unitDir, "manifest.yaml")
	if err := os.WriteFile(manifestPath, []byte("apiVersion: v1\nkind: Pod\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := WriteKubeUnit("site", manifestPath); err != nil {
		t.Fatal(err)
	}

	verifyUnitWithSystemd(t, GetUnitPath("site"))
}
