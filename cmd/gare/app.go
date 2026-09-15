package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/FacileStudio/gare/internal/builder"
	"github.com/FacileStudio/gare/internal/storage"
	"github.com/spf13/cobra"
)

type appCreateOptions struct {
	repo          string
	domain        string
	port          int
	branch        string
	appType       string
	containerfile string
	contextDir    string
	staticDir     string
	buildCmd      string
}

// NewAppCmd builds the app command group.
func NewAppCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "app",
		Short: "Manage applications",
	}

	cmd.AddCommand(newAppCreateCmd())
	return cmd
}

func newAppCreateCmd() *cobra.Command {
	opts := appCreateOptions{}
	cmd := &cobra.Command{
		Use:   "create <name> --repo <git-url> --domain <domain>",
		Short: "Create a new application",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runCreateApp(c.Context(), args[0], opts)
		},
	}

	cmd.Flags().StringVarP(&opts.repo, "repo", "r", "", "Git repository URL")
	cmd.Flags().StringVarP(&opts.domain, "domain", "d", "", "Domain name")
	cmd.Flags().IntVarP(&opts.port, "port", "p", 0, "Port to allocate (0 for auto-discovery)")
	cmd.Flags().StringVarP(&opts.branch, "branch", "b", "main", "Git branch")
	cmd.Flags().StringVarP(&opts.appType, "type", "t", "", "Application type (container or static)")
	cmd.Flags().StringVarP(&opts.containerfile, "containerfile", "f", "", "Path to Containerfile/Dockerfile")
	cmd.Flags().StringVar(&opts.contextDir, "context", "", "Build context directory relative to repository root")
	cmd.Flags().StringVar(&opts.staticDir, "static", "", "Static assets directory to serve (relative to repository root)")
	cmd.Flags().StringVar(&opts.buildCmd, "build-cmd", "", "Command to run during build/deployment")

	if err := cmd.MarkFlagRequired("repo"); err != nil {
		return cmd
	}
	if err := cmd.MarkFlagRequired("domain"); err != nil {
		return cmd
	}
	return cmd
}

func runCreateApp(parentCtx context.Context, name string, opts appCreateOptions) error {
	if err := validateCreateInputs(name, opts); err != nil {
		return err
	}
	opts.appType = strings.ToLower(opts.appType)

	ctx, cancel := context.WithTimeout(parentCtx, 5*time.Minute)
	defer cancel()

	baseDir := storage.DefaultBaseDir()
	appDir := storage.GetAppDir(baseDir, name)
	if _, err := os.Stat(storage.GetConfigPath(appDir)); err == nil {
		return fmt.Errorf("app %q already exists at %s", name, appDir)
	}

	if err := cloneAppRepo(ctx, appDir, opts); err != nil {
		return err
	}

	resolvedOpts, err := resolveAppOptions(appDir, opts)
	if err != nil {
		return err
	}

	if resolvedOpts.appType == "static" {
		return writeStaticArtifacts(name, appDir, resolvedOpts)
	}

	port, err := storage.DiscoverAvailablePort(baseDir, resolvedOpts.port)
	if err != nil {
		return fmt.Errorf("failed to discover port: %w", err)
	}
	resolvedOpts.port = port

	return writeAppArtifacts(name, appDir, resolvedOpts)
}

func validateCreateInputs(name string, opts appCreateOptions) error {
	if err := storage.ValidateAppName(name); err != nil {
		return err
	}
	if opts.repo == "" {
		return fmt.Errorf("--repo is required")
	}
	if opts.domain == "" {
		return fmt.Errorf("--domain is required")
	}
	appType := strings.ToLower(opts.appType)
	if appType != "" && appType != "container" && appType != "static" {
		return fmt.Errorf("invalid app type %q: must be container or static", opts.appType)
	}
	if strings.ContainsAny(opts.domain, " \t\r\n{}#;\"'\\/`$") {
		return fmt.Errorf("invalid domain %q: contains disallowed characters", opts.domain)
	}
	domainPattern := `^(\*\.)?([a-zA-Z0-9]([a-zA-Z0-9-_]{0,61}[a-zA-Z0-9])?\.)*` +
		`[a-zA-Z0-9]([a-zA-Z0-9-_]{0,61}[a-zA-Z0-9])?(:[0-9]{1,5})?$`
	domainRe := regexp.MustCompile(domainPattern)
	if !domainRe.MatchString(opts.domain) {
		return fmt.Errorf("invalid domain %q: must be a valid domain or hostname", opts.domain)
	}
	return nil
}

func cloneAppRepo(ctx context.Context, appDir string, opts appCreateOptions) error {
	repoDir := storage.GetRepoDir(appDir)
	printInfo(fmt.Sprintf("Cloning %s (%s) into %s...", opts.repo, opts.branch, repoDir))
	cloneOpts := builder.CloneOptions{
		RepoURL:   opts.repo,
		Branch:    opts.branch,
		TargetDir: repoDir,
		Stdout:    os.Stdout,
		Stderr:    os.Stderr,
	}
	return builder.Clone(ctx, cloneOpts)
}

func resolveAppOptions(appDir string, opts appCreateOptions) (appCreateOptions, error) {
	repoDir := storage.GetRepoDir(appDir)
	gf, err := storage.LoadGareFile(repoDir)
	if err != nil {
		return opts, fmt.Errorf("failed to load gare configuration: %w", err)
	}

	resolved := mergeGareFileDefaults(opts, gf)
	if resolved.staticDir != "" && resolved.appType == "" {
		resolved.appType = "static"
	}
	if resolved.appType == "" {
		resolved.appType = "container"
	}
	if resolved.appType == "static" && resolved.staticDir == "" {
		resolved.staticDir = "."
	}
	return resolved, nil
}
