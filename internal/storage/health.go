package storage

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/FacileStudio/gare/internal/health"
)

// HealthSection holds the health configuration of an application.
type HealthSection struct {
	Path   string         `yaml:"path,omitempty" json:"path,omitempty"`
	Probes []health.Probe `yaml:"probes,omitempty" json:"probes,omitempty"`
}

type legacyHealthConfig struct {
	Healthcheck string `json:"healthcheck,omitempty"`
}

// migrateHealth moves the legacy healthcheck key into the health section.
func migrateHealth(cfg *AppConfig, data []byte) {
	if cfg.HealthPath() != "" {
		return
	}
	var legacy legacyHealthConfig
	if err := json.Unmarshal(data, &legacy); err != nil || legacy.Healthcheck == "" {
		return
	}
	cfg.SetHealth(legacy.Healthcheck, nil)
}

// HealthPath returns the primary health check path if configured.
func (c *AppConfig) HealthPath() string {
	if c == nil || c.Health == nil {
		return ""
	}
	return c.Health.Path
}

// HealthProbes returns the configured probes, defaulting to the primary port and path.
func (c *AppConfig) HealthProbes() []health.Probe {
	if c == nil {
		return nil
	}
	if c.Health != nil && len(c.Health.Probes) > 0 {
		return c.Health.Probes
	}
	if c.Port <= 0 {
		return nil
	}
	return []health.Probe{{Name: c.Name, Port: c.Port, Path: c.HealthPath()}}
}

// SetHealth stores the primary health path and any explicit probes, clearing them when empty.
func (c *AppConfig) SetHealth(path string, probes []health.Probe) {
	if path == "" && len(probes) == 0 {
		c.Health = nil
		return
	}
	c.Health = &HealthSection{Path: path, Probes: probes}
}

// ValidateHealthProbes validates probe ports and paths.
func ValidateHealthProbes(probes []health.Probe) error {
	for _, probe := range probes {
		if probe.Port < 1 || probe.Port > 65535 {
			return fmt.Errorf("invalid health probe %s: port %d must be between 1 and 65535", probe.Label(), probe.Port)
		}
		if strings.ContainsAny(probe.Path, " \t\r\n") {
			return fmt.Errorf("invalid health probe %s: path %q must not contain whitespace", probe.Label(), probe.Path)
		}
		if strings.ContainsAny(probe.Name, "\r\n") {
			return fmt.Errorf("invalid health probe name %q: must not contain newlines", probe.Name)
		}
	}
	return nil
}
