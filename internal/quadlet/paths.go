package quadlet

import (
	"os"
	"path/filepath"
)

const (
	kubeExtension      = ".kube"
	containerExtension = ".container"
)

// Dir returns the rootless Quadlet source directory for the current user.
func Dir() string {
	if configHome := os.Getenv("XDG_CONFIG_HOME"); configHome != "" {
		return filepath.Join(configHome, "containers", "systemd")
	}
	return filepath.Join(homeDir(), ".config", "containers", "systemd")
}

// KubePath returns the Quadlet source path supervising an application pod manifest.
func KubePath(name string) string {
	return filepath.Join(Dir(), name+kubeExtension)
}

// ContainerPath returns the Quadlet source path serving an application container.
func ContainerPath(name string) string {
	return filepath.Join(Dir(), name+containerExtension)
}

func homeDir() string {
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		return home
	}
	return os.Getenv("HOME")
}
