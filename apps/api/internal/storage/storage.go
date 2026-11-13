package storage

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/FacileStudio/gare/apps/api/internal/atomicfile"
)

// ComposeRoute is how a compose app is reached: the compose file it runs and, when set, the service
// gare publishes the routed port for through its override file. AppConfig embeds it, so config.json
// keeps its flat compose_file and service keys.
type ComposeRoute struct {
	ComposeFile string `json:"compose_file,omitempty"`
	Service     string `json:"service,omitempty"`
}

// AppConfig represents an application's configuration and deployment metadata.
type AppConfig struct {
	Name          string   `json:"name"`
	RepoURL       string   `json:"repo_url"`
	Domains       []string `json:"domains,omitempty"`
	Port          int      `json:"port"`
	ContainerPort int      `json:"container_port,omitempty"`
	Branch        string   `json:"branch"`
	CreatedAt     string   `json:"created_at"`
	AppType       string   `json:"app_type,omitempty"`
	ComposeRoute
	Containerfile string         `json:"containerfile,omitempty"`
	ContextDir    string         `json:"context_dir,omitempty"`
	StaticDir     string         `json:"static_dir,omitempty"`
	BuildCmd      string         `json:"build_cmd,omitempty"`
	Health        *HealthSection `json:"health,omitempty"`
	Tags          []string       `json:"tags,omitempty"`
}

// GetConfigPath returns the config.json path within an app directory.
func GetConfigPath(appDir string) string {
	return filepath.Join(appDir, "config.json")
}

// saveConfig writes an AppConfig to the app directory's config.json atomically, readable by its
// owner alone.
func saveConfig(appDir string, cfg *AppConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return atomicfile.WriteFile(GetConfigPath(appDir), data, 0600)
}

// loadConfig reads an AppConfig from the app directory's config.json.
func loadConfig(appDir string) (*AppConfig, error) {
	data, err := os.ReadFile(GetConfigPath(appDir))
	if err != nil {
		return nil, err
	}
	var cfg AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	migrateDomain(&cfg, data)
	migrateHealth(&cfg, data)
	return &cfg, nil
}
