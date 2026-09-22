package main

import (
	"context"
	"fmt"
	"os"

	"charm.land/fang/v2"
	"github.com/FacileStudio/gare/cmd/gare/config"
	"github.com/spf13/cobra"
)

type rootFlags struct {
	verbose          bool
	configPath       string
	gitProvider      string
	useGitHubCLI     bool
	useGitLabCLI     bool
	credentialHelper string
	noColor          bool
}

// Execute runs the root command with fang styling and version handling.
func Execute(version string) error {
	return fang.Execute(context.Background(), newRootCmd(version), fang.WithVersion(version))
}

// NewRootCmd builds the root gare command and registers subcommands.
func NewRootCmd(version string) *cobra.Command {
	return newRootCmd(version)
}

func newRootCmd(version string) *cobra.Command {
	root := &cobra.Command{
		Use:           "gare",
		Version:       version,
		SilenceErrors: true,
		Short:         "Zero-daemon deployment CLI and GitOps orchestrator for Podman + Kubernetes workloads",
		Long: "gare is a minimalist, self-contained application manager that embraces\n" +
			"native Linux primitives for Podman and Kubernetes workloads:\n" +
			"- Zero Docker: uses podman kube play natively\n" +
			"- Quadlet units: container and static workloads become systemd user units through Quadlet\n" +
			"- Direct units: compose workloads keep gare-synthesized units\n" +
			"- Caddy ingress: drop-in snippets under /etc/caddy/conf.d/\n" +
			"- Stateless & file-driven: state stored purely on the filesystem",
	}
	root.SetVersionTemplate("{{.Name}} {{.Version}}\n")
	flags := &rootFlags{configPath: config.DefaultConfigPath()}
	registerRootFlags(root, flags)
	setupPersistentPreRun(root, flags)
	registerSubcommands(root)
	return root
}

func registerRootFlags(root *cobra.Command, flags *rootFlags) {
	root.PersistentFlags().BoolVarP(&flags.verbose, "verbose", "v", false, "Enable verbose logging")
	root.PersistentFlags().StringVarP(&flags.configPath, "config", "c", flags.configPath, "Path to config file (default: ~/.gare.yml)")
	root.PersistentFlags().StringVar(&flags.gitProvider, "git-provider", flags.gitProvider, "Override git provider (github, gitlab)")
	root.PersistentFlags().BoolVar(&flags.useGitHubCLI, "use-github-cli", flags.useGitHubCLI, "Force use of GitHub CLI")
	root.PersistentFlags().BoolVar(&flags.useGitLabCLI, "use-gitlab-cli", flags.useGitLabCLI, "Force use of GitLab CLI")
	root.PersistentFlags().StringVar(&flags.credentialHelper, "credential-helper", flags.credentialHelper, "Override credential helper")
	root.PersistentFlags().BoolVar(&flags.noColor, "no-color", false, "Disable colored output")
}

func setupPersistentPreRun(root *cobra.Command, flags *rootFlags) {
	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if err := applyColorPreference(flags.noColor); err != nil {
			return err
		}
		cfg := NewConfig()
		cfg.loader.SetVerbose(flags.verbose)
		cfg.loader.SetConfigPath(flags.configPath)
		cfg.loader.SetGitProvider(flags.gitProvider)
		cfg.loader.SetUseGitHubCLI(flags.useGitHubCLI)
		cfg.loader.SetUseGitLabCLI(flags.useGitLabCLI)
		cfg.loader.SetCredentialHelper(flags.credentialHelper)
		if _, err := cfg.loader.Load(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to load configuration: %v\n", err)
		}
		return nil
	}
}

func registerSubcommands(root *cobra.Command) {
	root.AddGroup(&cobra.Group{
		ID:    "management",
		Title: "MANAGEMENT COMMANDS",
	})
	root.AddGroup(&cobra.Group{
		ID:    "shortcuts",
		Title: "APPLICATION SHORTCUTS",
	})

	registerManagementCommands(root)
	registerShortcutCommands(root)
}

func registerManagementCommands(root *cobra.Command) {
	appCmd := NewAppCmd()
	appCmd.GroupID = "management"
	domainCmd := NewDomainCmd()
	domainCmd.GroupID = "management"
	envCmd := NewEnvCmd()
	envCmd.GroupID = "management"
	initCmd := NewInitCmd()
	initCmd.GroupID = "management"
	serverCmd := NewServerCmd()
	serverCmd.GroupID = "management"
	tagCmd := NewTagCmd()
	tagCmd.GroupID = "management"

	root.AddCommand(appCmd, domainCmd, envCmd, initCmd, serverCmd, tagCmd)
}

func registerShortcutCommands(root *cobra.Command) {
	shortcuts := []*cobra.Command{
		NewDeployCmd(),
		NewListCmd(),
		NewStatusCmd(),
		NewLogsCmd(),
		NewStartCmd(),
		NewStopCmd(),
		NewRestartCmd(),
		NewDestroyCmd(),
	}
	for _, cmd := range shortcuts {
		cmd.GroupID = "shortcuts"
		root.AddCommand(cmd)
	}
}
