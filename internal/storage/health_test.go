package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func writeLegacyConfig(appDir, content string) error {
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return err
	}
	return os.WriteFile(GetConfigPath(appDir), []byte(content), 0644)
}

func TestAppConfigHealthProbes(t *testing.T) {
	var nilCfg *AppConfig
	if got := nilCfg.HealthProbes(); got != nil {
		t.Errorf("nil HealthProbes: got %+v, want nil", got)
	}
	if got := nilCfg.HealthPath(); got != "" {
		t.Errorf("nil HealthPath: got %q, want empty", got)
	}

	fallback := &AppConfig{Name: "web", Port: 8100, Health: &HealthSection{Path: "/health"}}
	probes := fallback.HealthProbes()
	if len(probes) != 1 || probes[0].Port != 8100 || probes[0].Path != "/health" || probes[0].Name != "web" {
		t.Errorf("fallback probes: got %+v", probes)
	}

	explicit := &AppConfig{Name: "web", Port: 8100, Health: &HealthSection{
		Path:   "/health",
		Probes: []HealthProbe{{Name: "api", Port: 8100, Path: "/health"}, {Name: "db", Port: 8200}},
	}}
	if got := explicit.HealthProbes(); len(got) != 2 || got[1].Port != 8200 {
		t.Errorf("explicit probes: got %+v", got)
	}

	portless := &AppConfig{Name: "site", AppType: "static"}
	if got := portless.HealthProbes(); got != nil {
		t.Errorf("probes without a port: got %+v, want nil", got)
	}
}

func TestSetHealthClearsWhenEmpty(t *testing.T) {
	cfg := &AppConfig{Name: "web", Port: 8100, Health: &HealthSection{Path: "/health"}}
	cfg.SetHealth("", nil)
	if cfg.Health != nil {
		t.Errorf("SetHealth with empty values must clear the section, got %+v", cfg.Health)
	}
	cfg.SetHealth("/readyz", []HealthProbe{{Port: 8100, Path: "/readyz"}})
	if cfg.HealthPath() != "/readyz" || len(cfg.Health.Probes) != 1 {
		t.Errorf("SetHealth did not store values: %+v", cfg.Health)
	}
}

func TestMigrateLegacyHealthcheck(t *testing.T) {
	appDir := filepath.Join(t.TempDir(), "legacy")
	if err := writeLegacyConfig(appDir, `{"name":"legacy","port":8000,"healthcheck":"/healthz"}`); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(appDir)
	if err != nil {
		t.Fatalf("failed to load legacy config: %v", err)
	}
	if cfg.HealthPath() != "/healthz" {
		t.Errorf("legacy healthcheck was not migrated: %+v", cfg.Health)
	}
	if probes := cfg.HealthProbes(); len(probes) != 1 || probes[0].Path != "/healthz" {
		t.Errorf("unexpected probes after migration: %+v", probes)
	}
}

func TestValidateHealthProbes(t *testing.T) {
	if err := ValidateHealthProbes([]HealthProbe{{Name: "api", Port: 8100, Path: "/health"}}); err != nil {
		t.Errorf("valid probe rejected: %v", err)
	}
	invalid := [][]HealthProbe{
		{{Port: 0}},
		{{Port: 70000}},
		{{Port: 8100, Path: "/he alth"}},
		{{Port: 8100, Name: "bad\nname"}},
	}
	for _, probes := range invalid {
		if err := ValidateHealthProbes(probes); err == nil {
			t.Errorf("expected error for probes %+v", probes)
		}
	}
}

func TestHealthProbeLabel(t *testing.T) {
	if got := (HealthProbe{Name: "api", Port: 8100}).Label(); got != "api" {
		t.Errorf("Label with name: got %q", got)
	}
	if got := (HealthProbe{Port: 8100}).Label(); got != "port 8100" {
		t.Errorf("Label without name: got %q", got)
	}
}
