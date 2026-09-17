package config

// SetVerbose enables verbose mode.
func (l *Loader) SetVerbose(verbose bool) {
	l.config.Verbose = verbose
	l.changed["verbose"] = true
}

// SetConfigPath overrides the config file path.
func (l *Loader) SetConfigPath(path string) {
	l.config.ConfigPath = path
	l.changed["config_path"] = true
}

// SetGitProvider overrides the git provider.
func (l *Loader) SetGitProvider(provider string) {
	l.config.GitProvider = provider
	l.changed["git_provider"] = true
}

// SetUseGitHubCLI forces GitHub CLI usage.
func (l *Loader) SetUseGitHubCLI(use bool) {
	l.config.UseGitHubCLI = use
	l.changed["use_github_cli"] = true
}

// SetUseGitLabCLI forces GitLab CLI usage.
func (l *Loader) SetUseGitLabCLI(use bool) {
	l.config.UseGitLabCLI = use
	l.changed["use_gitlab_cli"] = true
}

// SetCredentialHelper overrides the credential helper.
func (l *Loader) SetCredentialHelper(helper string) {
	l.config.CredentialHelper = helper
	l.changed["credential_helper"] = true
}
