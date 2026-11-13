package workload

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/FacileStudio/gare/apps/api/internal/compose"
	"github.com/FacileStudio/gare/apps/api/internal/health"
	"github.com/FacileStudio/gare/apps/api/internal/podman"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
	"github.com/FacileStudio/gare/apps/api/internal/systemd"
)

// maxProbeSummary caps how many derived probes a deploy names before it counts the rest.
const maxProbeSummary = 4

// resolveComposeFile resolves the compose file to run and validates the routed port.
func resolveComposeFile(repoDir, configured string, port int) (string, error) {
	composeFile, err := compose.LocateFile(repoDir, configured)
	if err != nil {
		return "", err
	}
	if err := compose.ValidatePort(filepath.Join(repoDir, composeFile), port); err != nil {
		return "", err
	}
	return composeFile, nil
}

// ProjectName derives a stable per-app compose project name, so every app owns its own project
// rather than all of them sharing the name of their `repo` checkout.
func ProjectName(name string) string {
	return strings.ToLower(name)
}

// writeComposeUnit writes the synthesized unit for a compose stack and hands the stack over to it.
// Its guard checks the unit file without probing the podman workload runtime the container and
// static units need, because a compose unit runs through the podman compose provider rather than
// kube play, and it retires the Quadlet sources a previous workload type left behind so no stale
// source keeps generating a unit of the same name.
func writeComposeUnit(ctx context.Context, app App, composeFile string) error {
	name := app.Config.Name
	previous, err := claimUnit(ctx, name, false)
	if err != nil {
		return err
	}
	unitData := systemd.ComposeUnitData{
		Name:        name,
		RepoDir:     app.RepoDir(),
		ComposeFile: composeFile,
		ProjectName: ProjectName(name),
		EnvFile:     storage.GetAppEnvPath(app.Dir),
	}
	return handOver(ctx, app, systemd.ComposeUnitDescription(name), previous,
		func() error { return systemd.WriteComposeUnit(unitData) })
}

// verifyContainersGone warns when compose containers of the app's project outlived a stop or a
// teardown, so the operator learns about a stack still running behind a stopped unit.
func verifyContainersGone(ctx context.Context, app App, action string) {
	name := app.Config.Name
	leftovers, err := podman.ComposeProjectContainers(ctx, ProjectName(name))
	if err != nil || len(leftovers) == 0 {
		return
	}
	app.Report.Warn(fmt.Sprintf("%d compose container(s) still present after %s for %s; inspect with: podman ps -a",
		len(leftovers), action, name))
}

// warnComposeReachability tells the operator why a compose workload is not reachable from outside
// the host: its published port is bound to loopback and no domain routes to it, so only ingress can
// serve it. A compose app that publishes on every interface is left alone because it is reachable
// directly.
func warnComposeReachability(app App) {
	cfg := app.Config
	if len(cfg.Domains) > 0 {
		return
	}
	composeFile, err := compose.LocateFile(app.RepoDir(), cfg.ComposeFile)
	if err != nil {
		return
	}
	loopback, err := compose.LoopbackOnly(filepath.Join(app.RepoDir(), composeFile))
	if err != nil || !loopback {
		return
	}
	app.Report.Warn(fmt.Sprintf(
		"Compose app %s publishes port %d on loopback only and has no domain, so it is reachable "+
			"only from this host; add one with `gare domain add %s <hostname>` to serve it externally",
		cfg.Name, cfg.Port, cfg.Name))
}

// probeSummary names the first few probes and counts the rest, so a long list stays one line.
func probeSummary(probes []health.Probe) string {
	labels := make([]string, 0, len(probes))
	for _, probe := range probes {
		labels = append(labels, probe.Label())
	}
	if len(labels) > maxProbeSummary {
		return fmt.Sprintf("%s and %d more", strings.Join(labels[:maxProbeSummary], ", "), len(labels)-maxProbeSummary)
	}
	return strings.Join(labels, ", ")
}
