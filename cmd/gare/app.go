package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/FacileStudio/gare/internal/atomicfile"
	"github.com/FacileStudio/gare/internal/builder"
	"github.com/FacileStudio/gare/internal/caddy"
	"github.com/FacileStudio/gare/internal/storage"
	"github.com/FacileStudio/gare/internal/systemd"
	"github.com/spf13/cobra"
)

type appCreateOptions struct {
	repo   string
	domain string
	port   int
	branch string
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

	ctx, cancel := context.WithTimeout(parentCtx, 5*time.Minute)
	defer cancel()

	baseDir := storage.DefaultBaseDir()
	appDir := storage.GetAppDir(baseDir, name)
	if _, err := os.Stat(storage.GetConfigPath(appDir)); err == nil {
		return fmt.Errorf("app %q already exists at %s", name, appDir)
	}

	port, err := storage.DiscoverAvailablePort(baseDir, opts.port)
	if err != nil {
		return fmt.Errorf("failed to discover port: %w", err)
	}

	if err := cloneAppRepo(ctx, appDir, opts); err != nil {
		return err
	}

	return writeAppArtifacts(name, appDir, port, opts)
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

func writeAppArtifacts(name, appDir string, port int, opts appCreateOptions) error {
	manifestPath := storage.GetManifestPath(appDir)
	if err := resolveManifest(name, appDir, manifestPath, port); err != nil {
		return err
	}
	if err := systemd.WriteUnit(name, manifestPath); err != nil {
		return fmt.Errorf("failed to write systemd unit: %w", err)
	}
	if err := caddy.WriteSnippet(caddy.DefaultConfDir, name, opts.domain, port); err != nil {
		printWarning(fmt.Sprintf("Could not write Caddy snippet (%v)", err))
	}
	return saveAppMetadata(name, appDir, port, opts)
}

func resolveManifest(name, appDir, manifestPath string, port int) error {
	repoManifest := filepath.Join(storage.GetRepoDir(appDir), "manifest.yaml")
	if data, err := os.ReadFile(repoManifest); err == nil {
		if err := atomicfile.WriteFile(manifestPath, data, 0644); err != nil {
			return fmt.Errorf("failed to copy manifest: %w", err)
		}
		return nil
	}
	return storage.GenerateDefaultManifest(name, port, manifestPath)
}

func saveAppMetadata(name, appDir string, port int, opts appCreateOptions) error {
	appCfg := &storage.AppConfig{
		Name:      name,
		RepoURL:   opts.repo,
		Domain:    opts.domain,
		Port:      port,
		Branch:    opts.branch,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	if err := storage.SaveConfig(appDir, appCfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	printSuccess(fmt.Sprintf("App %q successfully created on port %d (%s)", name, port, opts.domain))
	return nil
}
