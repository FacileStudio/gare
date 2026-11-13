package cli

import (
	"context"
	"time"

	"github.com/FacileStudio/gare/apps/api/internal/appconfig"
	"github.com/FacileStudio/gare/apps/api/internal/engine"
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

// runCreateApp creates an application through the engine. The engine reports each step, the
// created app included, so nothing is left to print here.
func runCreateApp(parentCtx context.Context, name string, opts appCreateOptions) error {
	ctx, cancel := context.WithTimeout(parentCtx, 5*time.Minute)
	defer cancel()
	_, err := newEngine(ctx).Create(ctx, engine.CreateRequest{
		Name:   name,
		Repo:   opts.repo,
		Branch: opts.branch,
		Flags:  opts.flags(),
	})
	return err
}

// flags returns the create flags in the shape appconfig.Resolve ranks above gare.yml.
func (o appCreateOptions) flags() appconfig.Flags {
	return appconfig.Flags{
		Type:          o.appType,
		ComposeFile:   o.composeFile,
		Containerfile: o.containerfile,
		Context:       o.contextDir,
		StaticDir:     o.staticDir,
		BuildCmd:      o.buildCmd,
		Healthcheck:   o.healthcheck,
		Port:          o.port,
		ContainerPort: o.containerPort,
		Tags:          o.tags,
	}
}
