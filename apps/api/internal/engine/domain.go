package engine

import (
	"context"
	"fmt"

	"github.com/FacileStudio/gare/apps/api/internal/storage"
)

// AddDomain routes a hostname to an app, refusing one another app already owns, and serves it at
// once when the app is running.
func (e *Engine) AddDomain(ctx context.Context, name, hostname string) error {
	unlock, cfg, err := e.lockAndLoad(ctx, name)
	if err != nil {
		return err
	}
	defer unlock()
	if err := e.checkDomainAvailable(name, hostname); err != nil {
		return err
	}
	if err := cfg.AddDomain(hostname); err != nil {
		return err
	}
	if err := e.store.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	e.report(name).Done(fmt.Sprintf("Added domain %q to app %q", storage.NormalizeDomain(hostname), name))
	return e.syncIngressIfActive(ctx, cfg)
}

// RemoveDomain stops routing a hostname to an app, at once when the app is running.
func (e *Engine) RemoveDomain(ctx context.Context, name, hostname string) error {
	unlock, cfg, err := e.lockAndLoad(ctx, name)
	if err != nil {
		return err
	}
	defer unlock()
	if err := cfg.RemoveDomain(hostname); err != nil {
		return err
	}
	if err := e.store.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	e.report(name).Done(fmt.Sprintf("Removed domain %q from app %q", storage.NormalizeDomain(hostname), name))
	return e.syncIngressIfActive(ctx, cfg)
}

// checkDomainAvailable rejects an invalid hostname and one any app, this one included, already owns.
func (e *Engine) checkDomainAvailable(name, hostname string) error {
	if err := storage.ValidateDomain(hostname); err != nil {
		return err
	}
	apps, err := e.store.List()
	if err != nil {
		return fmt.Errorf("failed to list apps: %w", err)
	}
	if owner := storage.AppByDomain(apps, hostname); owner != nil {
		if owner.Name == name {
			return fmt.Errorf("app %q already owns domain %q", name, storage.NormalizeDomain(hostname))
		}
		return fmt.Errorf("domain %q is already in use by app %q", storage.NormalizeDomain(hostname), owner.Name)
	}
	return nil
}
