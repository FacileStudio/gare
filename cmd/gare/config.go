package main

import (
	"time"

	"github.com/FacileStudio/gare/cmd/gare/config"
)

// Config holds global configuration options grouped by concern.
type Config struct {
	Default *DefaultSection `yaml:"default"`
	Git     *GitSection     `yaml:"git"`
	Builder *BuilderSection `yaml:"builder"`
	Server  *ServerSection  `yaml:"server"`
	loader  *config.Loader
}

// DefaultSection holds default configuration values applied as fallbacks.
type DefaultSection struct {
	Verbose          bool   `yaml:"verbose"`
	ConfigPath       string `yaml:"config_path"`
	GitProvider      string `yaml:"git_provider"`
	UseGitHubCLI     bool   `yaml:"use_github_cli"`
	UseGitLabCLI     bool   `yaml:"use_gitlab_cli"`
	CredentialHelper string `yaml:"credential_helper"`
	SSHKeyPath       string `yaml:"ssh_key_path"`
}

// GitSection contains git provider and authentication settings.
type GitSection struct {
	Provider     string `yaml:"provider"`
	SkipAuth     bool   `yaml:"skip_auth"`
	ForceGitSSH  bool   `yaml:"force_git_ssh"`
	UseNativeGit bool   `yaml:"use_native_git"`
}

// BuilderSection contains container build settings.
type BuilderSection struct {
	Runtime      string        `yaml:"runtime"`
	DefaultArgs  []string      `yaml:"default_args"`
	BuildTimeout time.Duration `yaml:"build_timeout"`
}

// ServerSection contains HTTP server settings.
type ServerSection struct {
	WebhookSecret string `yaml:"webhook_secret"`
	PortRange     [2]int `yaml:"port_range"`
	BindAddr      string `yaml:"bind_addr"`
}

// NewConfig creates a Config with all default values applied.
func NewConfig() *Config {
	return &Config{
		Default: &DefaultSection{
			GitProvider:      "github",
			UseGitHubCLI:     true,
			UseGitLabCLI:     false,
			CredentialHelper: "git-credential-manager",
		},
		Git: &GitSection{
			Provider:     "github",
			UseNativeGit: true,
		},
		Builder: &BuilderSection{
			Runtime:      "podman",
			DefaultArgs:  []string{"--format=docker"},
			BuildTimeout: 30 * time.Minute,
		},
		Server: &ServerSection{
			PortRange: [2]int{8000, 65535},
			BindAddr:  "localhost:8080",
		},
		loader: config.NewLoader(),
	}
}