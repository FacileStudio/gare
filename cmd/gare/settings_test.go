package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func settingsCommand(t *testing.T, args ...string) (*cobra.Command, *rootFlags) {
	t.Helper()
	root := &cobra.Command{Use: "gare"}
	flags := &rootFlags{}
	registerRootFlags(root, flags)
	if err := root.ParseFlags(args); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}
	return root, flags
}

func writeSettingsFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "gare.yml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadSettingsReadsConfigFile(t *testing.T) {
	cfgPath := writeSettingsFile(t, "verbose: true\ngit_provider: gitlab\nuse_github_cli: true\ncredential_helper: \"!store\"\n")
	root, flags := settingsCommand(t, "--config", cfgPath)

	resolved, err := loadSettings(root, flags)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resolved.verbose {
		t.Error("verbose from the config file must reach the settings")
	}
	if resolved.auth.Provider != "gitlab" || !resolved.auth.UseGitHubCLI || resolved.auth.CredentialHelper != "!store" {
		t.Errorf("config file values must reach git auth, got %+v", resolved.auth)
	}
}

func TestLoadSettingsFlagOverridesFile(t *testing.T) {
	cfgPath := writeSettingsFile(t, "git_provider: gitlab\nuse_github_cli: true\n")
	root, flags := settingsCommand(t, "--config", cfgPath, "--git-provider", "github", "--use-github-cli=false")

	resolved, err := loadSettings(root, flags)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.auth.Provider != "github" {
		t.Errorf("--git-provider must override the file, got %q", resolved.auth.Provider)
	}
	if resolved.auth.UseGitHubCLI {
		t.Error("--use-github-cli=false must override a true value in the file")
	}
}

func TestLoadSettingsRejectsUnknownProvider(t *testing.T) {
	cfgPath := writeSettingsFile(t, "git_provider: githubb\n")
	root, flags := settingsCommand(t, "--config", cfgPath)

	if _, err := loadSettings(root, flags); err == nil {
		t.Error("expected an error for an unknown git provider in the config file")
	}
}

func TestLoadSettingsWithoutConfigFile(t *testing.T) {
	root, flags := settingsCommand(t, "--config", filepath.Join(t.TempDir(), "absent.yml"))

	resolved, err := loadSettings(root, flags)
	if err != nil {
		t.Fatalf("a missing config file must not fail the command: %v", err)
	}
	if resolved.verbose || resolved.auth.Provider != "" {
		t.Errorf("expected zero settings without a config file, got %+v", resolved)
	}
}
