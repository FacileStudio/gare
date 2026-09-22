package systemd

import (
	"strings"
	"testing"
)

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
