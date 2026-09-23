package systemd

import (
	"strings"
	"testing"
)

func composeUnitInput(name string) ComposeUnitData {
	base := "/home/user/.local/share/gare/apps/" + name
	return ComposeUnitData{
		Name:        name,
		PodmanPath:  "/usr/bin/podman",
		RepoDir:     base + "/repo",
		ComposeFile: "docker-compose.yml",
		ProjectName: name,
		EnvFile:     base + "/env",
	}
}

func TestGenerateComposeUnit(t *testing.T) {
	unit, err := GenerateComposeUnit(composeUnitInput("my-app"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedSnippets := []string{
		"Description=Gare Managed Compose App: my-app",
		"After=podman-user-wait-network-online.service podman.socket",
		"Wants=podman-user-wait-network-online.service",
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

func TestGenerateComposeUnitWaitsForNetworkInAUserSession(t *testing.T) {
	unit, err := GenerateComposeUnit(composeUnitInput("my-app"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(unit, "network-online.target") {
		t.Errorf("the user manager has no network-online.target, so ordering on it is silently dropped, got:\n%s", unit)
	}
	if !strings.Contains(unit, "Wants=podman-user-wait-network-online.service") ||
		!strings.Contains(unit, "After=podman-user-wait-network-online.service") {
		t.Errorf("compose unit must wait for the user-session network bridge, got:\n%s", unit)
	}
}

func TestGenerateComposeUnitResolvesPodmanPath(t *testing.T) {
	data := composeUnitInput("my-app")
	data.PodmanPath = ""
	unit, err := GenerateComposeUnit(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "ExecStart=" + ResolvePodmanPath() + " compose -f docker-compose.yml -p my-app up -d"
	if !strings.Contains(unit, expected) {
		t.Errorf("expected unit to contain %q, got:\n%s", expected, unit)
	}
}
