package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/FacileStudio/gare/internal/atomicfile"
	"github.com/FacileStudio/gare/internal/xdg"
)

// AppConfig represents an application's configuration and deployment metadata.
type AppConfig struct {
	Name          string         `json:"name"`
	RepoURL       string         `json:"repo_url"`
	Domains       []string       `json:"domains,omitempty"`
	Port          int            `json:"port"`
	ContainerPort int            `json:"container_port,omitempty"`
	Branch        string         `json:"branch"`
	CreatedAt     string         `json:"created_at"`
	AppType       string         `json:"app_type,omitempty"`
	ComposeFile   string         `json:"compose_file,omitempty"`
	Containerfile string         `json:"containerfile,omitempty"`
	ContextDir    string         `json:"context_dir,omitempty"`
	StaticDir     string         `json:"static_dir,omitempty"`
	BuildCmd      string         `json:"build_cmd,omitempty"`
	Health        *HealthSection `json:"health,omitempty"`
	Tags          []string       `json:"tags,omitempty"`
}

// DefaultBaseDir returns the default directory path for app storage.
func DefaultBaseDir() string {
	return filepath.Join(xdg.DataHome(), "gare", "apps")
}

// GetAppDir returns the directory path for a named app.
func GetAppDir(baseDir, name string) string {
	if baseDir == "" {
		baseDir = DefaultBaseDir()
	}
	return filepath.Join(baseDir, name)
}

// GetConfigPath returns the config.json path within an app directory.
func GetConfigPath(appDir string) string {
	return filepath.Join(appDir, "config.json")
}

// SaveConfig writes an AppConfig to config.json atomically.
func SaveConfig(appDir string, cfg *AppConfig) error {
	configPath := appDir
	if !strings.HasSuffix(appDir, ".json") {
		configPath = filepath.Join(appDir, "config.json")
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return atomicfile.WriteFile(configPath, data, 0644)
}

// LoadConfig reads an AppConfig from config.json.
func LoadConfig(appDir string) (*AppConfig, error) {
	configPath := appDir
	if !strings.HasSuffix(appDir, ".json") {
		configPath = filepath.Join(appDir, "config.json")
	}
	data, err := os.ReadFile(configPath)
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

// ListApps returns all configured applications sorted by name.
func ListApps(baseDir string) ([]*AppConfig, error) {
	if baseDir == "" {
		baseDir = DefaultBaseDir()
	}
	entries, err := os.ReadDir(baseDir)
	if os.IsNotExist(err) {
		return []*AppConfig{}, nil
	}
	if err != nil {
		return nil, err
	}
	var apps []*AppConfig
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		cfg, err := LoadConfig(filepath.Join(baseDir, entry.Name()))
		if err != nil {
			continue
		}
		apps = append(apps, cfg)
	}
	sort.Slice(apps, func(i, j int) bool {
		return apps[i].Name < apps[j].Name
	})
	return apps, nil
}

// DeleteAppStorage removes the application directory and all its contents.
func DeleteAppStorage(appDir string) error {
	return os.RemoveAll(appDir)
}
