package storage

import (
	"fmt"
	"net"
	"path/filepath"
	"testing"
)

func TestDiscoverAvailablePort(t *testing.T) {
	tmpDir := t.TempDir()
	port, err := DiscoverAvailablePort(tmpDir, 0)
	if err != nil || port < 8000 {
		t.Fatalf("DiscoverAvailablePort failed: port=%d err=%v", port, err)
	}
	testPreferredPort(t, tmpDir, port)

	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatalf("failed to listen on port %d: %v", port, err)
	}
	defer ln.Close()

	testBusyPort(t, tmpDir, port)
	testAssignedPort(t, tmpDir, port)
}

func testPreferredPort(t *testing.T, tmpDir string, port int) {
	chosen, err := DiscoverAvailablePort(tmpDir, port)
	if err != nil || chosen != port {
		t.Fatalf("DiscoverAvailablePort preferred got %d err %v", chosen, err)
	}
}

func testBusyPort(t *testing.T, tmpDir string, port int) {
	if _, err := DiscoverAvailablePort(tmpDir, port); err == nil {
		t.Errorf("expected error for busy port %d, got nil", port)
	}
}

func testAssignedPort(t *testing.T, tmpDir string, busyPort int) {
	nextPort, err := DiscoverAvailablePort(tmpDir, 0)
	if err != nil || nextPort == busyPort {
		t.Fatalf("expected different next port: %v", err)
	}
	appDir := filepath.Join(tmpDir, "existingapp")
	if err := SaveConfig(appDir, &AppConfig{Name: "existingapp", Port: nextPort}); err != nil {
		t.Fatal(err)
	}
	if _, err := DiscoverAvailablePort(tmpDir, nextPort); err == nil {
		t.Errorf("expected error for assigned app port %d, got nil", nextPort)
	}
	skipped, err := DiscoverAvailablePort(tmpDir, 0)
	if err != nil || skipped == nextPort || skipped == busyPort {
		t.Errorf("unexpected skipped port %d: %v", skipped, err)
	}
}
