package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDomainHostPort(t *testing.T) {
	tests := map[string]struct {
		host string
		port int
	}{
		"example.com":              {"example.com", 0},
		"sub.example.com":          {"sub.example.com", 0},
		"localhost":                {"localhost", 0},
		"localhost:8080":           {"localhost", 8080},
		"myapp.example.com:443":    {"myapp.example.com", 443},
		"broken.example.com:0":     {"broken.example.com:0", 0},
		"broken.example.com:99999": {"broken.example.com:99999", 0},
		"broken.example.com:port":  {"broken.example.com:port", 0},
	}
	for hostname, want := range tests {
		host, port := DomainHostPort(hostname)
		if host != want.host || port != want.port {
			t.Errorf("DomainHostPort(%q) = (%q, %d), want (%q, %d)", hostname, host, port, want.host, want.port)
		}
	}
}

func TestValidateDomain(t *testing.T) {
	valid := []string{
		"example.com",
		"sub.example.com",
		"*.example.com",
		"localhost",
		"localhost:8080",
		"myapp.example.com:443",
	}
	for _, tc := range valid {
		if err := ValidateDomain(tc); err != nil {
			t.Errorf("expected %q to be valid, got: %v", tc, err)
		}
	}

	invalid := []string{
		"example.com { reverse_proxy }",
		"example.com\nnewline",
		"bad;injection",
		"domain/path",
		"has space.com",
		"{injection}",
		"bad#comment",
		"",
	}
	for _, tc := range invalid {
		if err := ValidateDomain(tc); err == nil {
			t.Errorf("expected %q to be invalid", tc)
		}
	}
}

func TestNormalizeDomain(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"  Example.COM  ", "example.com"},
		{"*.Example.COM", "*.example.com"},
		{"LOCALHOST", "localhost"},
	}
	for _, tc := range cases {
		got := NormalizeDomain(tc.input)
		if got != tc.want {
			t.Errorf("NormalizeDomain(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestAddDomain(t *testing.T) {
	cfg := &AppConfig{Name: "myapp"}
	if err := cfg.AddDomain("Example.COM"); err != nil {
		t.Fatalf("AddDomain failed: %v", err)
	}
	if err := cfg.AddDomain("example.com"); err == nil {
		t.Error("expected duplicate domain to fail")
	}
	if len(cfg.Domains) != 1 {
		t.Errorf("expected 1 domain, got %d", len(cfg.Domains))
	}
	if cfg.Domains[0] != "example.com" {
		t.Errorf("expected normalized domain, got %q", cfg.Domains[0])
	}
}

func TestRemoveDomain(t *testing.T) {
	cfg := &AppConfig{Name: "myapp", Domains: []string{"a.com", "b.com", "c.com"}}
	if err := cfg.RemoveDomain("B.com"); err != nil {
		t.Fatalf("RemoveDomain failed: %v", err)
	}
	if cfg.HasDomain("b.com") {
		t.Error("expected b.com to be removed")
	}
	if err := cfg.RemoveDomain("b.com"); err == nil {
		t.Error("expected removing missing domain to fail")
	}
	if len(cfg.Domains) != 2 {
		t.Errorf("expected 2 remaining domains, got %d", len(cfg.Domains))
	}
}

func TestHasDomain(t *testing.T) {
	cfg := &AppConfig{Domains: []string{"a.com", "B.com"}}
	if !cfg.HasDomain("b.com") {
		t.Error("expected case-insensitive match")
	}
	if cfg.HasDomain("c.com") {
		t.Error("expected c.com to not be found")
	}
}

func TestAppByDomain(t *testing.T) {
	apps := []*AppConfig{
		{Name: "alpha", Domains: []string{"alpha.example.com"}},
		{Name: "beta", Domains: []string{"beta.example.com", "www.beta.example.com"}},
	}
	if app := AppByDomain(apps, "beta.example.com"); app == nil || app.Name != "beta" {
		t.Errorf("expected to find beta, got %+v", app)
	}
	if app := AppByDomain(apps, "www.beta.example.com"); app == nil || app.Name != "beta" {
		t.Error("expected to find beta for www alias")
	}
	if app := AppByDomain(apps, "gamma.example.com"); app != nil {
		t.Error("expected nil for unknown domain")
	}
}

func TestLoadConfigMigratesLegacyDomain(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "myapp")
	configPath := GetConfigPath(appDir)

	legacyJSON := `{"name":"myapp","domain":"legacy.example.com","port":8000}`
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := writeTestFile(t, configPath, legacyJSON); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(appDir)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if len(cfg.Domains) != 1 || cfg.Domains[0] != "legacy.example.com" {
		t.Errorf("expected migrated domain, got %v", cfg.Domains)
	}
}

func TestSaveConfigDropsLegacyDomain(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "myapp")
	configPath := GetConfigPath(appDir)

	cfg := &AppConfig{Name: "myapp", Domains: []string{"new.example.com"}, Port: 8000}
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := SaveConfig(appDir, cfg); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	if _, ok := raw["domain"]; ok {
		t.Error("expected legacy domain key to be absent from saved JSON")
	}
	if _, ok := raw["domains"]; !ok {
		t.Error("expected domains key in saved JSON")
	}
}

func writeTestFile(t *testing.T, path, content string) error {
	t.Helper()
	return os.WriteFile(path, []byte(content), 0644)
}
