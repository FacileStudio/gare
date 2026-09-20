package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/FacileStudio/gare/internal/health"
	"github.com/FacileStudio/gare/internal/storage"
	"github.com/FacileStudio/gare/internal/systemd"
)

func verifyHealth(ctx context.Context, cfg *storage.AppConfig) error {
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
	if err := verifyHealthProbes(ctx, probes); err != nil {
		return err
	}
	printSuccess(fmt.Sprintf("%d health probe(s) passed", len(probes)))
	return nil
}

func verifyServiceActive(ctx context.Context, cfg *storage.AppConfig) error {
	time.Sleep(100 * time.Millisecond)
	props, _ := systemd.GetServiceProperties(ctx, cfg.Name)
	if props == nil || props.ActiveState == "active" {
		return nil
	}
	return fmt.Errorf("service %s is not active (%s/%s, result: %s); inspect with: gare logs %s",
		cfg.Name, props.ActiveState, props.SubState, props.Result, cfg.Name)
}

func verifyHealthProbes(ctx context.Context, probes []storage.HealthProbe) error {
	for _, probe := range probes {
		target := probeTarget(probe)
		printInfo(fmt.Sprintf("Verifying health probe %s at %s...", probe.Label(), target))
		if err := health.Probe(ctx, target, 30*time.Second); err != nil {
			return fmt.Errorf("health probe %s failed: %w", probe.Label(), err)
		}
	}
	return nil
}

func probeSummary(probes []storage.HealthProbe) string {
	labels := make([]string, 0, len(probes))
	for _, probe := range probes {
		labels = append(labels, probe.Label())
	}
	return strings.Join(labels, ", ")
}

func probeTarget(probe storage.HealthProbe) string {
	path := probe.Path
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return fmt.Sprintf("http://127.0.0.1:%d%s", probe.Port, path)
}
