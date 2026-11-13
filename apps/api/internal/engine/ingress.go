package engine

import (
	"context"
	"fmt"
	"strings"

	"github.com/FacileStudio/gare/apps/api/internal/caddy"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
	"github.com/FacileStudio/gare/apps/api/internal/systemd"
)

// syncAppIngress writes the app's Caddy snippet when it has domains and a port, and removes it
// otherwise.
func syncAppIngress(cfg *storage.AppConfig) error {
	confDir := caddy.ResolveConfDir()
	if len(cfg.Domains) > 0 && cfg.Port > 0 {
		return caddy.WriteSnippet(confDir, cfg.Name, cfg.Domains, cfg.Port)
	}
	return caddy.RemoveSnippet(confDir, cfg.Name)
}

// activateIngress writes or removes the app's snippet and makes the running server serve it. An app
// meant to be served publicly cannot be reported active while the server did not accept the
// configuration, so once it has domains a failure is fatal and the ingress is probed afterwards; an
// app reachable on its port alone only warns, because nothing depends on ingress.
func (e *Engine) activateIngress(ctx context.Context, cfg *storage.AppConfig) error {
	report := e.report(cfg.Name)
	if err := syncAppIngress(cfg); err != nil {
		return ingressFailure(report, cfg, fmt.Errorf("could not sync the Caddy snippet for %s: %w", cfg.Name, err))
	}
	if len(cfg.Domains) > 0 {
		report.Verbose(fmt.Sprintf("Wrote the Caddy snippet for %s: %s", cfg.Name, strings.Join(cfg.Domains, ", ")))
	} else {
		report.Verbose(fmt.Sprintf("Removed the Caddy snippet for %s (no domains configured)", cfg.Name))
	}
	if err := caddy.Reload(ctx); err != nil {
		return ingressFailure(report, cfg, fmt.Errorf("caddy did not accept the configuration for %s: %w", cfg.Name, err))
	}
	return verifyIngress(ctx, report, cfg)
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
// effect without a redeploy. A stopped app picks the change up at its next start.
func (e *Engine) syncIngressIfActive(ctx context.Context, cfg *storage.AppConfig) error {
	status, err := systemd.IsActive(ctx, cfg.Name)
	if err != nil || status != "active" {
		return nil
	}
	if err := syncAppIngress(cfg); err != nil {
		return fmt.Errorf("failed to sync ingress: %w", err)
	}
	if err := caddy.Reload(ctx); err != nil {
		e.report(cfg.Name).Warn(fmt.Sprintf("Caddy reload returned error: %v", err))
	}
	return nil
}
