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
		"Delegate=yes",
		"Restart=on-failure",
		"RestartSec=5s",
		"TimeoutStopSec=70s",
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

	if strings.Contains(unit, "network-online.target") {
		t.Errorf("unit should not contain network-online.target, got:\n%s", unit)
	}
}

func TestResolvePodmanPath(t *testing.T) {
	path := ResolvePodmanPath()
	if path == "" {
		t.Fatal("expected non-empty podman path")
	}
}

func TestUserEnviron(t *testing.T) {
	env := userEnviron()
	hasRuntimeDir := false
	hasDBusBus := false
	for _, e := range env {
		if strings.HasPrefix(e, "XDG_RUNTIME_DIR=") {
			hasRuntimeDir = true
		}
		if strings.HasPrefix(e, "DBUS_SESSION_BUS_ADDRESS=") {
			hasDBusBus = true
		}
	}
	if !hasRuntimeDir {
		t.Error("expected XDG_RUNTIME_DIR to be present in userEnviron")
	}
	if !hasDBusBus {
		t.Error("expected DBUS_SESSION_BUS_ADDRESS to be present in userEnviron")
	}
}
