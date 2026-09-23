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

const maxProbeSummary = 4

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
	if err := verifyHealthProbes(ctx, probes, cfg.Name); err != nil {
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

func verifyHealthProbes(ctx context.Context, probes []health.Probe, appName string) error {
	for _, probe := range probes {
		target := probeTarget(probe)
		printInfo(fmt.Sprintf("Verifying health probe %s at %s...", probe.Label(), target))
		if err := health.Verify(ctx, target, 30*time.Second); err != nil {
			return fmt.Errorf("health probe %s failed — check the service with `gare logs %s`: %w", probe.Label(), appName, err)
		}
	}
	return nil
}

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

func probeTarget(probe health.Probe) string {
	path := probe.Path
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return fmt.Sprintf("http://127.0.0.1:%d%s", probe.Port, path)
}
