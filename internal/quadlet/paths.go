package quadlet

import (
	"path/filepath"

	"github.com/FacileStudio/gare/internal/xdg"
)

const (
	kubeExtension      = ".kube"
	containerExtension = ".container"
)

// Dir returns the rootless Quadlet source directory for the current user.
func Dir() string {
	return filepath.Join(xdg.ConfigHome(), "containers", "systemd")
}

// KubePath returns the Quadlet source path supervising an application pod manifest.
func KubePath(name string) string {
	return filepath.Join(Dir(), name+kubeExtension)
}

// ContainerPath returns the Quadlet source path serving an application container.
func ContainerPath(name string) string {
	return filepath.Join(Dir(), name+containerExtension)
}
