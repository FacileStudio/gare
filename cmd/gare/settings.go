package main

import (
	"context"

	"github.com/FacileStudio/gare/cmd/gare/config"
	"github.com/FacileStudio/gare/internal/git"
	"github.com/spf13/cobra"
)

// settings holds the global configuration resolved for one invocation. It rides on the command
// context rather than in a package variable, so the webhook daemon's deploy path resolves no
// settings at all and keeps git's built-in defaults.
type settings struct {
	verbose bool
	auth    git.Auth
}

type settingsKey struct{}

// withSettings returns a context carrying the resolved settings.
func withSettings(ctx context.Context, resolved settings) context.Context {
	return context.WithValue(ctx, settingsKey{}, resolved)
}

// settingsFrom returns the settings a command attached to ctx, or the zero value when there are none.
func settingsFrom(ctx context.Context) settings {
	resolved, ok := ctx.Value(settingsKey{}).(settings)
	if !ok {
		return settings{}
	}
	return resolved
}

// applyGlobalFlags marks every global flag the operator actually passed as a CLI override, so the
// config file still supplies the settings they left alone.
func applyGlobalFlags(loader *config.Loader, cmd *cobra.Command, flags *rootFlags) {
	changed := cmd.Flags().Changed
	if changed("config") {
		loader.SetConfigPath(flags.configPath)
	}
	if changed("verbose") {
		loader.SetVerbose(flags.verbose)
	}
	if changed("git-provider") {
		loader.SetGitProvider(flags.gitProvider)
	}
	if changed("use-github-cli") {
		loader.SetUseGitHubCLI(flags.useGitHubCLI)
	}
	if changed("use-gitlab-cli") {
		loader.SetUseGitLabCLI(flags.useGitLabCLI)
	}
	if changed("credential-helper") {
		loader.SetCredentialHelper(flags.credentialHelper)
	}
}

// loadSettings reads the config file and applies only the global flags the operator actually passed,
// so a value in ~/.gare.yml survives unless the command line overrides it.
func loadSettings(cmd *cobra.Command, flags *rootFlags) (settings, error) {
	loader := config.NewLoader()
	applyGlobalFlags(loader, cmd, flags)
	cfg, err := loader.Load()
	if err != nil {
		return settings{}, err
	}
	resolved := settings{
		verbose: cfg.Verbose,
		auth: git.Auth{
			Provider:         cfg.GitProvider,
			UseGitHubCLI:     cfg.UseGitHubCLI,
			UseGitLabCLI:     cfg.UseGitLabCLI,
			CredentialHelper: cfg.CredentialHelper,
		},
	}
	return resolved, resolved.auth.Validate()
}
