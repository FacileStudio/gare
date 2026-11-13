package engine

import (
	"context"
	"fmt"
	"strings"

	"github.com/FacileStudio/gare/apps/api/internal/ingress"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
	"github.com/FacileStudio/gare/apps/api/internal/systemd"
)

// activateIngress makes the running server serve the app's snippet, or drop it, and probes the
// ingress. An app meant to be served publicly cannot be reported active while the server did not
// accept the configuration, so once it has domains a failure is fatal; an app reachable on its port
// alone only warns, because nothing depends on ingress.
func (e *Engine) activateIngress(ctx context.Context, cfg *storage.AppConfig) error {
	report := e.report(cfg.Name)
	if err := ingress.Apply(ctx, cfg); err != nil {
		return ingressFailure(report, cfg, err)
	}
	if len(cfg.Domains) == 0 {
		report.Verbose(fmt.Sprintf("Removed the Caddy snippet for %s (no domains configured)", cfg.Name))
		return nil
	}
	report.Verbose(fmt.Sprintf("Ingress answered for every domain of %s: %s", cfg.Name, strings.Join(cfg.Domains, ", ")))
	return nil
}

// ingressFailure keeps an ingress error fatal for an app meant to be served publicly and downgrades
// it to a warning for an app with no domains, because nothing depends on ingress then.
func ingressFailure(report reporter, cfg *storage.AppConfig, err error) error {
	if len(cfg.Domains) == 0 {
		report.Warn(err.Error())
		return nil
	}
	return err
}

// syncIngressIfActive rewrites the ingress of an app whose unit is running, so a domain change takes
// effect without a redeploy. A stopped app picks the change up at its next start. The domain change
// is already saved, so a refusal warns: the snippet was rolled back and the next start retries it.
func (e *Engine) syncIngressIfActive(ctx context.Context, cfg *storage.AppConfig) error {
	status, err := systemd.IsActive(ctx, cfg.Name)
	if err != nil || status != "active" {
		return nil
	}
	if err := ingress.Apply(ctx, cfg); err != nil {
		e.report(cfg.Name).Warn(err.Error())
	}
	return nil
}
