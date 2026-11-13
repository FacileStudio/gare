package engine

import (
	"context"

	"github.com/FacileStudio/gare/apps/api/internal/health"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
	"github.com/FacileStudio/gare/apps/api/internal/systemd"
)

// AppStatus holds detailed status for a single application. Its JSON shape is what
// `gare status --json` prints, so field names are part of the CLI's contract.
type AppStatus struct {
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
	Probes        []health.Probe             `json:"probes,omitempty"`
	Tags          []string                   `json:"tags,omitempty"`
	Service       *systemd.ServiceProperties `json:"service,omitempty"`
}

// Status collects an application's configuration, deployed commit and unit state. An app whose
// unit systemd does not report is inactive.
func (e *Engine) Status(ctx context.Context, name string) (*AppStatus, error) {
	cfg, err := e.Config(name)
	if err != nil {
		return nil, err
	}
	details := &AppStatus{
		Name:          cfg.Name,
		AppType:       cfg.AppType,
		Status:        "inactive",
		Domains:       cfg.Domains,
		Port:          cfg.Port,
		ContainerPort: cfg.ContainerPort,
		RepoURL:       cfg.RepoURL,
		Branch:        cfg.Branch,
		Commit:        DeployedCommit(ctx, storage.GetRepoDir(e.store.AppDir(name))),
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
	return details, nil
}
