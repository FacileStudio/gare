package storage

import (
	"fmt"
	"net"
)

// DiscoverAvailablePort finds an available TCP port starting from 8000.
func DiscoverAvailablePort(baseDir string, preferredPort int) (int, error) {
	assigned, err := collectAssignedPorts(baseDir)
	if err != nil {
		return 0, fmt.Errorf("cannot tell which ports are taken: %w", err)
	}
	if preferredPort > 0 {
		if !isPortAvailable(preferredPort, assigned) {
			return 0, fmt.Errorf("port %d is not available", preferredPort)
		}
		return preferredPort, nil
	}
	return scanFreePort(assigned)
}

// collectAssignedPorts returns the ports recorded by every app. It fails when an app's
// configuration cannot be read, because that app's port would otherwise be handed out again.
func collectAssignedPorts(baseDir string) (map[int]bool, error) {
	assigned := make(map[int]bool)
	if baseDir == "" {
		return assigned, nil
	}
	apps, err := ListAppsStrict(baseDir)
	if err != nil {
		return nil, err
	}
	for _, app := range apps {
		if app.Port > 0 {
			assigned[app.Port] = true
		}
	}
	return assigned, nil
}

func isPortAvailable(port int, assigned map[int]bool) bool {
	if assigned[port] {
		return false
	}
	return isPortFree(port)
}

func isPortFree(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	defer func() { _ = ln.Close() }()
	return true
}

func scanFreePort(assigned map[int]bool) (int, error) {
	for port := 8000; port <= 65535; port++ {
		if isPortAvailable(port, assigned) {
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available port found")
}
