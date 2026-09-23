package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/FacileStudio/gare/internal/git"
	"github.com/FacileStudio/gare/internal/storage"
	"github.com/FacileStudio/gare/internal/systemd"
	"github.com/spf13/cobra"
)

type appCreateOptions struct {
	repo          string
	port          int
	containerPort int
	branch        string
	appType       string
	composeFile   string
	containerfile string
	contextDir    string
	staticDir     string
	buildCmd      string
	healthcheck   string
	healthProbes  []storage.HealthProbe
	tags          []string
}

// NewAppCmd builds the app command group.
func NewAppCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "app",
		Aliases: []string{"apps"},
		Short:   "Manage applications",
		Long: "Manage application workloads and lifecycle.\n" +
			"Subcommands cover creating, deploying, inspecting, and tearing down applications.",
		Example: `  gare app create myapp --repo https://github.com/org/repo.git
  gare app deploy myapp
  gare app list
  gare app status myapp
  gare app logs myapp -f
  gare app destroy myapp`,
	}

	cmd.AddCommand(newAppCreateCmd())
	cmd.AddCommand(NewDeployCmd())
	cmd.AddCommand(NewListCmd())
	cmd.AddCommand(NewStatusCmd())
	cmd.AddCommand(NewStartCmd())
	cmd.AddCommand(NewStopCmd())
	cmd.AddCommand(NewRestartCmd())
	cmd.AddCommand(NewLogsCmd())
	cmd.AddCommand(NewDestroyCmd())
	cmd.AddCommand(NewDomainCmd())
	cmd.AddCommand(NewEnvCmd())
	cmd.AddCommand(NewTagCmd())
	return cmd
}

func newAppCreateCmd() *cobra.Command {
	opts := appCreateOptions{}
	cmd := &cobra.Command{
		Use:   "create [name] --repo <git-url>",
		Short: "Create a new application",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			name := ""
			if len(args) > 0 {
				name = args[0]
			}
			return runCreateApp(c.Context(), name, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.repo, "repo", "r", "", "Git repository URL")
	cmd.Flags().IntVarP(&opts.port, "port", "p", 0, "Port to allocate (0 for auto-discovery)")
	cmd.Flags().IntVar(&opts.containerPort, "container-port", 0, "Container internal port (from Containerfile EXPOSE)")
	cmd.Flags().StringVarP(&opts.branch, "branch", "b", "main", "Git branch")
	cmd.Flags().StringVarP(&opts.appType, "type", "t", "", "Application type (container, static, or compose)")
	cmd.Flags().StringVar(&opts.composeFile, "compose-file", "", "Compose file to run for compose apps")
	cmd.Flags().StringVarP(&opts.containerfile, "containerfile", "f", "", "Path to Containerfile/Dockerfile")
	cmd.Flags().StringVar(&opts.contextDir, "context", "", "Build context directory relative to repository root")
	cmd.Flags().StringVar(&opts.staticDir, "static", "", "Static assets directory to serve (relative to repository root)")
	cmd.Flags().StringVar(&opts.buildCmd, "build-cmd", "", "Command to run during build/deployment")
	cmd.Flags().StringVar(&opts.healthcheck, "healthcheck", "", "Health check HTTP path (e.g. /health)")
	cmd.Flags().StringSliceVar(&opts.tags, "tag", nil, "Application tags")

	if err := cmd.MarkFlagRequired("repo"); err != nil {
		return cmd
	}
	return cmd
}

func runCreateApp(parentCtx context.Context, name string, opts appCreateOptions) error {
	if name == "" {
		name = deriveAppName(opts.repo)
	}
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
		if rmErr := storage.DeleteAppStorage(appDir); rmErr != nil {
			return err
		}
		return err
	}

	return setupAppWorkload(ctx, baseDir, name, appDir, opts)
}

func deriveAppName(repoURL string) string {
	cleaned := strings.TrimSpace(repoURL)
	cleaned = strings.TrimSuffix(cleaned, "/")
	cleaned = strings.TrimSuffix(cleaned, ".git")
	if idx := strings.LastIndexAny(cleaned, "/:"); idx != -1 {
		cleaned = cleaned[idx+1:]
	}
	return strings.ToLower(cleaned)
}

func setupAppWorkload(ctx context.Context, baseDir, name, appDir string, opts appCreateOptions) error {
	resolvedOpts, err := resolveAppOptions(appDir, opts)
	if err != nil {
		return err
	}
	port, err := storage.DiscoverAvailablePort(baseDir, resolvedOpts.port)
	if err != nil {
		return fmt.Errorf("failed to discover port: %w", err)
	}
	resolvedOpts.port = port
	if err := writeWorkloadArtifacts(ctx, name, appDir, resolvedOpts); err != nil {
		return err
	}
	if err := systemd.DaemonReload(ctx); err != nil {
		printWarning(fmt.Sprintf("daemon-reload error: %v", err))
	}
	return nil
}

func validateCreateInputs(name string, opts appCreateOptions) error {
	if name == "" {
		return fmt.Errorf("app name is required as an argument or inferrable from --repo")
	}
	if err := storage.ValidateAppName(name); err != nil {
		return err
	}
	if opts.repo == "" {
		return fmt.Errorf("--repo is required")
	}
	if opts.containerPort < 0 || opts.containerPort > 65535 {
		return fmt.Errorf("container port must be between 1 and 65535, got %d", opts.containerPort)
	}
	for _, tag := range opts.tags {
		if err := storage.ValidateTag(tag); err != nil {
			return err
		}
	}
	return nil
}

func cloneAppRepo(ctx context.Context, appDir string, opts appCreateOptions) error {
	repoDir := storage.GetRepoDir(appDir)
	printInfo(fmt.Sprintf("Cloning %s (%s) into %s...", opts.repo, opts.branch, repoDir))
	cloneOpts := git.CloneOptions{
		RepoURL:   opts.repo,
		Branch:    opts.branch,
		TargetDir: repoDir,
		Auth:      settingsFrom(ctx).auth,
		Stdout:    os.Stdout,
		Stderr:    os.Stderr,
	}
	return git.Clone(ctx, cloneOpts)
}

func resolveAppOptions(appDir string, opts appCreateOptions) (appCreateOptions, error) {
	repoDir := storage.GetRepoDir(appDir)
	gf, err := storage.LoadGareFile(repoDir)
	if err != nil {
		return opts, fmt.Errorf("failed to load gare configuration: %w", err)
	}

	resolved, err := mergeGareFileDefaults(opts, gf)
	if err != nil {
		return opts, err
	}
	return normalizeCreateOptions(repoDir, resolved)
}
