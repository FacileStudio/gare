package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/FacileStudio/gare/internal/caddy"
	"github.com/FacileStudio/gare/internal/storage"
	"github.com/FacileStudio/gare/internal/systemd"
)

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
func activateIngress(ctx context.Context, cfg *storage.AppConfig) error {
	if err := syncAppIngress(cfg); err != nil {
		return ingressFailure(cfg, fmt.Errorf("could not sync the Caddy snippet for %s: %w", cfg.Name, err))
	}
	if len(cfg.Domains) > 0 {
		printVerbose(ctx, "Wrote the Caddy snippet for %s: %s", cfg.Name, strings.Join(cfg.Domains, ", "))
	} else {
		printVerbose(ctx, "Removed the Caddy snippet for %s (no domains configured)", cfg.Name)
	}
	if err := caddy.Reload(ctx); err != nil {
		return ingressFailure(cfg, fmt.Errorf("caddy did not accept the configuration for %s: %w", cfg.Name, err))
	}
	return verifyIngress(ctx, cfg)
}

// ingressFailure keeps an ingress error fatal for an app meant to be served publicly and downgrades
// it to a warning for an app with no domains, because nothing depends on ingress then.
func ingressFailure(cfg *storage.AppConfig, err error) error {
	if len(cfg.Domains) == 0 {
		printWarning(err.Error())
		return nil
	}
	return err
}

func syncAndReloadIfActive(ctx context.Context, cfg *storage.AppConfig) error {
	status, err := systemd.IsActive(ctx, cfg.Name)
	if err == nil && status == "active" {
		if err := syncAppIngress(cfg); err != nil {
			return fmt.Errorf("failed to sync ingress: %w", err)
		}
		if err := caddy.Reload(ctx); err != nil {
			printWarning(fmt.Sprintf("Caddy reload returned error: %v", err))
		}
	}
	return nil
}
