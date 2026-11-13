package workload

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/FacileStudio/gare/apps/api/internal/systemd"
)

// claimUnit fails before anything is written or stopped when a unit file gare does not own already
// occupies the application's unit name, or, when probePodman is set, when the host cannot run the
// unit at all. A compose unit skips the probe because it runs through the compose provider rather
// than kube play. It returns the description of the gare unit that already occupies the name, empty
// when the name is free, which is what tells a workload type change from a plain redeploy.
func claimUnit(ctx context.Context, name string, probePodman bool) (string, error) {
	previous, err := guardUnitFile(name)
	if err != nil || !probePodman {
		return previous, err
	}
	return previous, systemd.CheckPodmanWorkloads(ctx)
}

// handOver writes one workload's unit and hands the workload over to it. Every workload type runs
// this same sequence so it cannot drift between them: the write comes first, so a failure to write
// leaves a workload that was already running fine untouched, and the workload being replaced is
// stopped only afterwards, while systemd still serves the definition it loaded, so the outgoing
// unit's own ExecStop tears its workload down rather than a teardown that names one it never
// started.
func handOver(ctx context.Context, app App, description, previous string, write func() error) error {
	name := app.Config.Name
	if err := write(); err != nil {
		return fmt.Errorf("failed to write systemd unit: %w", err)
	}
	app.Report.Verbose(fmt.Sprintf("Wrote systemd unit %s", systemd.GetUnitPath(name)))
	retireChangedWorkload(ctx, app, previous, description)
	return retireLegacyQuadletWorkload(ctx, app)
}

// guardUnitFile refuses to overwrite a unit file gare did not write, so an operator's own unit is
// never replaced by a deploy. It returns the description of the gare unit occupying the name, which
// is how a caller tells a workload type change from a plain redeploy, or empty when the name is
// free.
func guardUnitFile(name string) (string, error) {
	unitPath := systemd.GetUnitPath(name)
	data, err := os.ReadFile(unitPath)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to inspect %s: %w", unitPath, err)
	}
	for _, description := range systemd.GareUnitDescriptions(name) {
		if strings.Contains(string(data), "Description="+description+"\n") {
			return description, nil
		}
	}
	return "", fmt.Errorf("unit file %s already exists and was not written by gare — remove it and deploy again, "+
		"or run `gare destroy %s` to remove the whole app", unitPath, name)
}

// retireChangedWorkload stops the outgoing workload when a deploy changes an application's workload
// type, so its own teardown runs. systemd keeps serving the definition it loaded until
// daemon-reload, so stopping after the unit file is rewritten still runs the outgoing unit's
// ExecStop: a compose stack goes down through `podman compose down` rather than being replaced by a
// `kube down` that names no pod, which would leave the stack running in front of the new unit.
func retireChangedWorkload(ctx context.Context, app App, previous, current string) {
	if !replacedWorkloadType(previous, current) {
		return
	}
	name := app.Config.Name
	app.Report.Step(fmt.Sprintf("Retiring the previous workload for %s before its replacement takes over...", name))
	if err := systemd.Stop(ctx, name); err != nil {
		app.Report.Warn(fmt.Sprintf("systemctl stop returned error: %v", err))
	}
}

// replacedWorkloadType reports whether a deploy replaces the workload gare already supervises with
// one of a different type. A redeploy of the same type keeps its workload running, because the
// restart that follows reuses it rather than tearing it down and building it again.
func replacedWorkloadType(previous, current string) bool {
	return previous != "" && previous != current
}

// retireLegacyQuadletWorkload retires the Quadlet workload an older gare handed to the Podman
// generator. The unit is stopped while its own definition is still loaded, because once the source
// is gone the generated definition disappears with it and its own teardown never runs, leaving an
// orphaned pod or container in front of the unit gare now writes itself.
func retireLegacyQuadletWorkload(ctx context.Context, app App) error {
	name := app.Config.Name
	exists, err := systemd.LegacyQuadletSourcesExist(name)
	if err != nil || !exists {
		return err
	}
	app.Report.Step(fmt.Sprintf("Retiring the Quadlet workload for %s before the synthesized unit takes over...", name))
	if err := systemd.Stop(ctx, name); err != nil {
		app.Report.Warn(fmt.Sprintf("systemctl stop returned error: %v", err))
	}
	if err := systemd.RemoveLegacyQuadletSources(name); err != nil {
		return fmt.Errorf("failed to retire the Quadlet sources: %w", err)
	}
	return nil
}
