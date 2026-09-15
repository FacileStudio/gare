package systemd

import (
	"strings"
	"testing"
)

func TestGenerateUnit(t *testing.T) {
	unit, err := GenerateUnit("my-app", "/home/user/.local/share/gare/apps/my-app/manifest.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedSnippets := []string{
		"Description=Gare Managed App: my-app",
		"Type=exec",
		"KillMode=mixed",
		"Restart=on-failure",
		"RestartSec=5s",
		"kube play --replace -w /home/user/.local/share/gare/apps/my-app/manifest.yaml",
		"ExecStopPost=-/usr/bin/podman kube down /home/user/.local/share/gare/apps/my-app/manifest.yaml",
		"SyslogIdentifier=%N",
		"WantedBy=default.target",
	}

	for _, snippet := range expectedSnippets {
		if !strings.Contains(unit, snippet) {
			t.Errorf("expected unit to contain %q, got:\n%s", snippet, unit)
		}
	}
}

func TestResolvePodmanPath(t *testing.T) {
	path := ResolvePodmanPath()
	if path == "" {
		t.Fatal("expected non-empty podman path")
	}
}
