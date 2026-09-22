package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// GareConfig represents the global configuration that can be set via file or CLI flags.
type GareConfig struct {
	Verbose          bool      `yaml:"verbose"`
	ConfigPath       string    `yaml:"config_path"`
	GitProvider      string    `yaml:"git_provider"`
	UseGitHubCLI     bool      `yaml:"use_github_cli"`
	UseGitLabCLI     bool      `yaml:"use_gitlab_cli"`
	CredentialHelper string    `yaml:"credential_helper"`
	LoadedAt         time.Time `yaml:"-" json:"-"`
}

// Loader holds the configuration and provides methods to load and override it.
type Loader struct {
	config  *GareConfig
	changed map[string]bool
}

// NewLoader returns a new Loader with default values.
func NewLoader() *Loader {
	return &Loader{
		config: &GareConfig{
			ConfigPath: DefaultConfigPath(),
			LoadedAt:   time.Now(),
		},
		changed: make(map[string]bool),
	}
}

// DefaultConfigPath returns the default path for the configuration file.
func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "~/.gare.yml"
	}
	return filepath.Join(home, ".gare.yml")
}

func expandHomePath(path string) string {
	if len(path) >= 2 && path[0:2] == "~/" {
		if home := os.Getenv("HOME"); home != "" {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

func (l *Loader) mergeFile(fileCfg *GareConfig) {
	if !l.changed["verbose"] {
		l.config.Verbose = fileCfg.Verbose
	}
	if !l.changed["config_path"] {
		l.config.ConfigPath = fileCfg.ConfigPath
	}
	if !l.changed["git_provider"] && fileCfg.GitProvider != "" {
		l.config.GitProvider = fileCfg.GitProvider
	}
	if !l.changed["use_github_cli"] {
		l.config.UseGitHubCLI = fileCfg.UseGitHubCLI
	}
	if !l.changed["use_gitlab_cli"] {
		l.config.UseGitLabCLI = fileCfg.UseGitLabCLI
	}
	if !l.changed["credential_helper"] && fileCfg.CredentialHelper != "" {
		l.config.CredentialHelper = fileCfg.CredentialHelper
	}
}

// Load reads the configuration from file (if it exists) and returns it.
func (l *Loader) Load() (*GareConfig, error) {
	configPath := expandHomePath(l.config.ConfigPath)
	data, err := os.ReadFile(configPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}
	if len(data) > 0 {
		var fileCfg GareConfig
		if err := yaml.Unmarshal(data, &fileCfg); err != nil {
			return nil, fmt.Errorf("failed to parse config file %s: %w", configPath, err)
		}
		l.mergeFile(&fileCfg)
	}
	l.config.LoadedAt = time.Now()
	return l.config, nil
}

// Get returns the current merged configuration.
func (l *Loader) Get() *GareConfig {
	return l.config
}
