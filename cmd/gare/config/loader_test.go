package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadGareConfigDefaults(t *testing.T) {
	loader := NewLoader()
	cfg, err := loader.Load()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if cfg.ConfigPath != DefaultConfigPath() {
		t.Errorf("expected default config path, got %s", cfg.ConfigPath)
	}
}

func TestLoadGareConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "gare.yml")
	yamlContent := "verbose: true\nconfig_path: ~/.custom_gare.yml\ngit_provider: \"gitlab\"\nuse_github_cli: false\nuse_gitlab_cli: true\ncredential_helper: \"cache\"\n"
	if err := os.WriteFile(yamlFile, []byte(yamlContent), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewLoader()
	loader.SetConfigPath(yamlFile)
	cfg, err := loader.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertLoadedValues(t, cfg)
}

func assertLoadedValues(t *testing.T, cfg *GareConfig) {
	if !cfg.Verbose || cfg.GitProvider != "gitlab" || cfg.UseGitHubCLI || !cfg.UseGitLabCLI || cfg.CredentialHelper != "cache" {
		t.Errorf("unexpected config loaded: %+v", cfg)
	}
}

func TestLoadGareConfigCLIOverrides(t *testing.T) {
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "gare.yml")
	if err := os.WriteFile(yamlFile, []byte("git_provider: gitlab\n"), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewLoader()
	loader.SetConfigPath(yamlFile)
	loader.SetGitProvider("github")

	cfg, err := loader.Load()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if cfg.GitProvider != "github" {
		t.Errorf("expected CLI override (github), got %s", cfg.GitProvider)
	}
}