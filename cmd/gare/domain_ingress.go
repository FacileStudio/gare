package main

import (
	"context"
	"fmt"

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
