package engine

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/FacileStudio/gare/apps/api/internal/health"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
	"github.com/FacileStudio/gare/apps/api/internal/systemd"
)

// verifyHealth checks the restarted unit is active, then that every health probe passes. An app
// with no port and no probe has nothing to check.
func (e *Engine) verifyHealth(ctx context.Context, cfg *storage.AppConfig) error {
	probes := cfg.HealthProbes()
	if cfg.Port <= 0 && len(probes) == 0 {
		return nil
	}
	if err := verifyServiceActive(ctx, cfg); err != nil {
		return err
	}
	if len(probes) == 0 {
		return nil
	}
	report := e.report(cfg.Name)
	if err := verifyHealthProbes(ctx, report, probes, cfg.Name); err != nil {
		return err
	}
	report.Done(fmt.Sprintf("%d health probe(s) passed", len(probes)))
	return nil
}

// verifyServiceActive fails when systemd reports the unit anything but active right after its
// restart, pointing at the logs that say why.
func verifyServiceActive(ctx context.Context, cfg *storage.AppConfig) error {
	time.Sleep(100 * time.Millisecond)
	props, _ := systemd.GetServiceProperties(ctx, cfg.Name)
	if props == nil || props.ActiveState == "active" {
		return nil
	}
	return fmt.Errorf("service %s is not active (%s/%s, result: %s); inspect with: gare logs %s",
		cfg.Name, props.ActiveState, props.SubState, props.Result, cfg.Name)
}

// verifyHealthProbes runs each probe in turn and names the first one that fails.
func verifyHealthProbes(ctx context.Context, report reporter, probes []health.Probe, appName string) error {
	for _, probe := range probes {
		target := ProbeTarget(probe)
		report.Step(fmt.Sprintf("Verifying health probe %s at %s...", probe.Label(), target))
		if err := health.Verify(ctx, probe, target, 30*time.Second); err != nil {
			return fmt.Errorf("health probe %s failed — check the service with `gare logs %s`: %w", probe.Label(), appName, err)
		}
	}
	return nil
}

// ProbeTarget returns the loopback URL a probe is sent to.
func ProbeTarget(probe health.Probe) string {
	path := probe.Path
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return fmt.Sprintf("http://127.0.0.1:%d%s", probe.Port, path)
}
