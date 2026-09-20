package storage

import (
	"encoding/json"
	"fmt"
	"strings"
)

// HealthProbe describes a single HTTP readiness probe for a workload.
type HealthProbe struct {
	Name string `yaml:"name,omitempty" json:"name,omitempty"`
	Port int    `yaml:"port" json:"port"`
	Path string `yaml:"path,omitempty" json:"path,omitempty"`
}

// HealthSection holds the health configuration of an application.
type HealthSection struct {
	Path   string        `yaml:"path,omitempty" json:"path,omitempty"`
	Probes []HealthProbe `yaml:"probes,omitempty" json:"probes,omitempty"`
}

type legacyHealthConfig struct {
	Healthcheck string `json:"healthcheck,omitempty"`
}

// Label returns a human readable identifier for the probe.
func (p HealthProbe) Label() string {
	if p.Name != "" {
		return p.Name
	}
	return fmt.Sprintf("port %d", p.Port)
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
func (c *AppConfig) HealthProbes() []HealthProbe {
	if c == nil {
		return nil
	}
	if c.Health != nil && len(c.Health.Probes) > 0 {
		return c.Health.Probes
	}
	if c.Port <= 0 {
		return nil
	}
	return []HealthProbe{{Name: c.Name, Port: c.Port, Path: c.HealthPath()}}
}

// SetHealth stores the primary health path and any explicit probes, clearing them when empty.
func (c *AppConfig) SetHealth(path string, probes []HealthProbe) {
	if path == "" && len(probes) == 0 {
		c.Health = nil
		return
	}
	c.Health = &HealthSection{Path: path, Probes: probes}
}

// ResolveHealthProbes returns the probes declared in the gare configuration.
func (g *GareFile) ResolveHealthProbes() []HealthProbe {
	if g == nil {
		return nil
	}
	return g.Healthchecks
}

// ValidateHealthProbes validates probe ports and paths.
func ValidateHealthProbes(probes []HealthProbe) error {
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
