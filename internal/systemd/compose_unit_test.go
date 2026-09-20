package systemd

import (
	"strings"
	"testing"
)

func TestGenerateComposeUnit(t *testing.T) {
	unit, err := GenerateComposeUnit(ComposeUnitData{
		Name:        "my-app",
		PodmanPath:  "/usr/bin/podman",
		RepoDir:     "/home/user/.local/share/gare/apps/my-app/repo",
		ComposeFile: "docker-compose.yml",
		ProjectName: "my-app",
		EnvFile:     "/home/user/.local/share/gare/apps/my-app/env",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedSnippets := []string{
		"Description=Gare Managed Compose App: my-app",
		"After=network-online.target podman.socket",
		"Requires=podman.socket",
		"Type=oneshot",
		"RemainAfterExit=yes",
		"WorkingDirectory=/home/user/.local/share/gare/apps/my-app/repo",
		"EnvironmentFile=-/home/user/.local/share/gare/apps/my-app/env",
		"TimeoutStartSec=300s",
		"ExecStart=/usr/bin/podman compose -f docker-compose.yml -p my-app up -d",
		"ExecStop=-/usr/bin/podman compose -f docker-compose.yml -p my-app down",
		"ExecStopPost=-/usr/bin/podman compose -f docker-compose.yml -p my-app down",
	}

	for _, snippet := range expectedSnippets {
		if !strings.Contains(unit, snippet) {
			t.Errorf("expected unit to contain %q, got:\n%s", snippet, unit)
		}
	}
	assertComposeUnitDetached(t, unit)
}

func assertComposeUnitDetached(t *testing.T, unit string) {
	t.Helper()
	if strings.Contains(unit, "Type=exec") {
		t.Errorf("compose unit must not attach to the provider process, got:\n%s", unit)
	}
	if strings.Contains(unit, "kube play") {
		t.Errorf("compose unit must not fall back to kube play, got:\n%s", unit)
	}
}

func TestGenerateComposeUnitResolvesPodmanPath(t *testing.T) {
	unit, err := GenerateComposeUnit(ComposeUnitData{
		Name:        "my-app",
		RepoDir:     "/srv/repo",
		ComposeFile: "compose.yml",
		ProjectName: "my-app",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "ExecStart=" + ResolvePodmanPath() + " compose -f compose.yml -p my-app up -d"
	if !strings.Contains(unit, expected) {
		t.Errorf("expected unit to contain %q, got:\n%s", expected, unit)
	}
}
