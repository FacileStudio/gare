package workload

import (
	"github.com/FacileStudio/gare/apps/api/internal/manifest"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
)

// EnvStore is where an application's environment variables actually live: the app env file a
// static or compose workload reads, or the pod manifest a container workload reads. Its operations
// are fields with distinct names, so an app env writer cannot be passed where a manifest writer
// belongs — the mix-up two identically typed closures invite.
type EnvStore struct {
	Label  string
	Read   func(appDir string) (map[string]string, error)
	Write  func(appDir string, vars map[string]string) error
	Remove func(appDir string, keys []string) error
}

// appEnvFileStore returns the store a static or compose workload reads.
func appEnvFileStore() EnvStore {
	return EnvStore{
		Label:  "app env file",
		Read:   storage.GetAppEnv,
		Write:  storage.SetAppEnv,
		Remove: storage.UnsetAppEnv,
	}
}

// podManifestStore returns the store a container workload reads.
func podManifestStore() EnvStore {
	return EnvStore{
		Label: "manifest",
		Read: func(appDir string) (map[string]string, error) {
			return manifest.GetEnv(storage.GetManifestPath(appDir))
		},
		Write: func(appDir string, vars map[string]string) error {
			return manifest.SetEnv(storage.GetManifestPath(appDir), vars)
		},
		Remove: func(appDir string, keys []string) error {
			return manifest.UnsetEnv(storage.GetManifestPath(appDir), keys)
		},
	}
}
