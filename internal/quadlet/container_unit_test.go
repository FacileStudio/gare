package quadlet

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateContainerUnit(t *testing.T) {
	content, err := GenerateContainerUnit(StaticSiteUnit(
		"my-site", 8100, "/srv/repo/dist", "/home/user/.local/share/gare/apps/my-site/env",
		"/home/user/.local/share/gare/apps/my-site/Caddyfile"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedSnippets := []string{
		"Description=Gare Managed Static App: my-site",
		"[Container]",
		"Image=docker.io/library/caddy:2-alpine",
		"ContainerName=my-site",
		"Entrypoint=caddy",
		"Exec=run --config /etc/caddy/Caddyfile --adapter caddyfile",
		"PublishPort=8100:80",
		"Volume=/srv/repo/dist:/srv:ro,Z",
		"Volume=/home/user/.local/share/gare/apps/my-site/Caddyfile:/etc/caddy/Caddyfile:ro,Z",
		"EnvironmentFile=/home/user/.local/share/gare/apps/my-site/env",
		"Restart=on-failure",
		"[Install]",
		"WantedBy=default.target",
	}
	for _, snippet := range expectedSnippets {
		if !strings.Contains(content, snippet) {
			t.Errorf("expected source to contain %q, got:\n%s", snippet, content)
		}
	}
	if strings.Contains(content, `Volume="`) || strings.Contains(content, `EnvironmentFile="`) {
		t.Errorf("quadlet does not re-quote these values, quoting breaks the generated mount:\n%s", content)
	}
}

func TestGenerateContainerUnitLeavesExecutionToQuadlet(t *testing.T) {
	content, err := GenerateContainerUnit(StaticSiteUnit("my-site", 8100, "/srv/dist", "/srv/env", "/srv/Caddyfile"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, forbidden := range []string{"ExecStart=", "ExecStop", "podman"} {
		if strings.Contains(content, forbidden) {
			t.Errorf("quadlet must own %q, got:\n%s", forbidden, content)
		}
	}
}

func TestGenerateContainerUnitRejectsBadInput(t *testing.T) {
	configFile := "/srv/Caddyfile"
	cases := map[string]ContainerUnitData{
		"relative root":      StaticSiteUnit("my-site", 8100, "dist", "/srv/env", configFile),
		"root with space":    StaticSiteUnit("my-site", 8100, "/srv/my dist", "/srv/env", configFile),
		"quoted root":        StaticSiteUnit("my-site", 8100, `/srv/"dist"`, "/srv/env", configFile),
		"relative env file":  StaticSiteUnit("my-site", 8100, "/srv/dist", "env", configFile),
		"relative caddyfile": StaticSiteUnit("my-site", 8100, "/srv/dist", "/srv/env", "Caddyfile"),
		"missing port":       StaticSiteUnit("my-site", 0, "/srv/dist", "/srv/env", configFile),
		"out of range port":  StaticSiteUnit("my-site", 70000, "/srv/dist", "/srv/env", configFile),
	}
	for name, data := range cases {
		if _, err := GenerateContainerUnit(data); err == nil {
			t.Errorf("expected an error for %s", name)
		}
	}
}

func TestWriteContainerUnitRetiresKubeSource(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := WriteKubeUnit("my-site", "/srv/manifest.yaml"); err != nil {
		t.Fatalf("WriteKubeUnit failed: %v", err)
	}

	if err := WriteContainerUnit(StaticSiteUnit("my-site", 8100, "/srv/dist", "/srv/env", "/srv/Caddyfile")); err != nil {
		t.Fatalf("WriteContainerUnit failed: %v", err)
	}
	if _, err := os.Stat(ContainerPath("my-site")); err != nil {
		t.Errorf("expected the container source to exist: %v", err)
	}
	if _, err := os.Stat(KubePath("my-site")); !os.IsNotExist(err) {
		t.Error("a workload type change must not leave two sources generating the same unit")
	}
}

func TestWriteKubeUnitRetiresContainerSource(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := WriteContainerUnit(StaticSiteUnit("my-site", 8100, "/srv/dist", "/srv/env", "/srv/Caddyfile")); err != nil {
		t.Fatalf("WriteContainerUnit failed: %v", err)
	}

	if err := WriteKubeUnit("my-site", "/srv/manifest.yaml"); err != nil {
		t.Fatalf("WriteKubeUnit failed: %v", err)
	}
	if _, err := os.Stat(KubePath("my-site")); err != nil {
		t.Errorf("expected the kube source to exist: %v", err)
	}
	if _, err := os.Stat(ContainerPath("my-site")); !os.IsNotExist(err) {
		t.Error("a workload type change must not leave two sources generating the same unit")
	}
}

func TestRemoveDeletesEverySource(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := os.MkdirAll(Dir(), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(KubePath("my-site"), []byte("[Kube]\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ContainerPath("my-site"), []byte("[Container]\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := Remove("my-site"); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}
	for _, path := range []string{KubePath("my-site"), ContainerPath("my-site")} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("expected %s to be removed", path)
		}
	}
}

func TestGeneratedContainerSourceIsAcceptedByQuadlet(t *testing.T) {
	unitDir := t.TempDir()
	source, err := GenerateContainerUnit(StaticSiteUnit("my-site", 8100, "/srv/dist", "/srv/env", "/srv/Caddyfile"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(unitDir, "my-site.container"), []byte(source), 0644); err != nil {
		t.Fatal(err)
	}

	generated := runQuadletDryRun(t, unitDir)
	expected := []string{
		"---my-site.service---",
		"--entrypoint caddy",
		"-v /srv/dist:/srv:ro,Z",
		"-v /srv/Caddyfile:/etc/caddy/Caddyfile:ro,Z",
		"--publish 8100:80",
		"--env-file /srv/env",
		"docker.io/library/caddy:2-alpine run --config /etc/caddy/Caddyfile --adapter caddyfile",
		"WantedBy=default.target",
	}
	for _, snippet := range expected {
		if !strings.Contains(generated, snippet) {
			t.Errorf("expected the generated unit to contain %q, got:\n%s", snippet, generated)
		}
	}
}
