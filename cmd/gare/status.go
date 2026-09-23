package main

import (
	"context"
	"io"
	"time"

	"github.com/FacileStudio/gare/internal/storage"
	"github.com/FacileStudio/gare/internal/systemd"
	"github.com/spf13/cobra"
)

type statusOptions struct {
	jsonOutput bool
}

// AppStatusDetails holds detailed status for a single application.
type AppStatusDetails struct {
	Name          string                     `json:"name"`
	AppType       string                     `json:"app_type"`
	Status        string                     `json:"status"`
	Domains       []string                   `json:"domains,omitempty"`
	Port          int                        `json:"port,omitempty"`
	ContainerPort int                        `json:"container_port,omitempty"`
	RepoURL       string                     `json:"repo_url"`
	Branch        string                     `json:"branch"`
	Commit        string                     `json:"commit"`
	CreatedAt     string                     `json:"created_at"`
	Healthcheck   string                     `json:"healthcheck,omitempty"`
	Probes        []storage.HealthProbe      `json:"probes,omitempty"`
	Tags          []string                   `json:"tags,omitempty"`
	Service       *systemd.ServiceProperties `json:"service,omitempty"`
}

// NewStatusCmd builds the status command.
func NewStatusCmd() *cobra.Command {
	opts := statusOptions{}
	cmd := &cobra.Command{
		Use:   "status <name>",
		Short: "Display detailed status for an application",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
			defer cancel()
			return runStatus(ctx, cmd.OutOrStdout(), args[0], opts)
		},
	}
	cmd.Flags().BoolVar(&opts.jsonOutput, "json", false, "Output JSON object")
	return cmd
}

func runStatus(ctx context.Context, w io.Writer, name string, opts statusOptions) error {
	if err := storage.ValidateAppName(name); err != nil {
		return err
	}
	baseDir := storage.DefaultBaseDir()
	appDir := storage.GetAppDir(baseDir, name)
	cfg, err := storage.LoadConfig(appDir)
	if err != nil {
		return appConfigError(name, err)
	}

	details := collectAppStatus(ctx, appDir, cfg)
	if opts.jsonOutput {
		return writeJSON(w, details)
	}
	return outputStatusHuman(w, details)
}

func collectAppStatus(ctx context.Context, appDir string, cfg *storage.AppConfig) *AppStatusDetails {
	repoDir := storage.GetRepoDir(appDir)
	details := &AppStatusDetails{
		Name:          cfg.Name,
		AppType:       cfg.AppType,
		Status:        "inactive",
		Domains:       cfg.Domains,
		Port:          cfg.Port,
		ContainerPort: cfg.ContainerPort,
		RepoURL:       cfg.RepoURL,
		Branch:        cfg.Branch,
		Commit:        deployedCommit(ctx, repoDir),
		CreatedAt:     cfg.CreatedAt,
		Healthcheck:   cfg.HealthPath(),
		Probes:        cfg.HealthProbes(),
		Tags:          cfg.Tags,
	}
	props, _ := systemd.GetServiceProperties(ctx, cfg.Name)
	details.Service = props
	if props != nil && props.ActiveState != "" {
		details.Status = props.ActiveState
	}
	return details
}
