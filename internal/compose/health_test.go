package compose

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/FacileStudio/gare/internal/health"
)

func TestHealthProbes(t *testing.T) {
	repoDir := t.TempDir()
	content := "services:\n" +
		"  web:\n" +
		"    ports:\n" +
		"      - \"18420:80\"\n" +
		"    healthcheck:\n" +
		"      test: [\"CMD-SHELL\", \"wget -q -O /dev/null http://localhost/health || exit 1\"]\n" +
		"  admin:\n" +
		"    ports:\n" +
		"      - \"18421:80\"\n" +
		"    healthcheck:\n" +
		"      test: [\"CMD\", \"curl\", \"-f\", \"http://localhost:80/\"]\n"
	writeComposeFile(t, repoDir, "compose.yml", content)

	probes, err := HealthProbes(filepath.Join(repoDir, "compose.yml"))
	if err != nil {
		t.Fatalf("HealthProbes failed: %v", err)
	}
	want := []health.Probe{
		{Name: "admin", Port: 18421, Path: "/"},
		{Name: "web", Port: 18420, Path: "/health"},
	}
	if !slices.Equal(probes, want) {
		t.Errorf("HealthProbes: got %+v, want %+v", probes, want)
	}
}

func TestHealthProbesSkipsUnprobeableServices(t *testing.T) {
	repoDir := t.TempDir()
	writeComposeFile(t, repoDir, "compose.yml", unprobeableComposeYAML())

	probes, err := HealthProbes(filepath.Join(repoDir, "compose.yml"))
	if err != nil {
		t.Fatalf("HealthProbes failed: %v", err)
	}
	want := []health.Probe{{Name: "hostport", Port: 18432, Path: "/readyz"}}
	if !slices.Equal(probes, want) {
		t.Errorf("only host-reachable http healthchecks must be probed: got %+v, want %+v", probes, want)
	}
}

func unprobeableComposeYAML() string {
	return "services:\n" +
		"  internal:\n" +
		"    healthcheck:\n" +
		"      test: [\"CMD-SHELL\", \"wget -q -O /dev/null http://localhost/health\"]\n" +
		"  db:\n" +
		"    ports:\n" +
		"      - \"15432:5432\"\n" +
		"    healthcheck:\n" +
		"      test: [\"CMD-SHELL\", \"test -f /data/marker\"]\n" +
		"  quiet:\n" +
		"    ports:\n" +
		"      - \"18430:80\"\n" +
		"  disabled:\n" +
		"    ports:\n" +
		"      - \"18431:80\"\n" +
		"    healthcheck:\n" +
		"      test: [\"NONE\"]\n" +
		"  hostport:\n" +
		"    ports:\n" +
		"      - \"18432:8080\"\n" +
		"    healthcheck:\n" +
		"      test: \"curl -f http://localhost:8080/readyz\"\n"
}

func TestHealthProbesLongSyntaxPorts(t *testing.T) {
	repoDir := t.TempDir()
	content := "services:\n" +
		"  api:\n" +
		"    ports:\n" +
		"      - target: 3000\n" +
		"        published: 18440\n" +
		"    healthcheck:\n" +
		"      test: [\"CMD-SHELL\", \"wget -q -O /dev/null http://127.0.0.1:3000/healthz\"]\n"
	writeComposeFile(t, repoDir, "compose.yml", content)

	probes, err := HealthProbes(filepath.Join(repoDir, "compose.yml"))
	if err != nil {
		t.Fatalf("HealthProbes failed: %v", err)
	}
	want := []health.Probe{{Name: "api", Port: 18440, Path: "/healthz"}}
	if !slices.Equal(probes, want) {
		t.Errorf("long syntax mapping: got %+v, want %+v", probes, want)
	}
}

func TestHealthProbesInvalidFile(t *testing.T) {
	repoDir := t.TempDir()
	if _, err := HealthProbes(filepath.Join(repoDir, "missing.yml")); err == nil {
		t.Error("expected error for a missing compose file")
	}
	writeComposeFile(t, repoDir, "broken.yml", "services:\n  web:\n    healthcheck: [")
	if _, err := HealthProbes(filepath.Join(repoDir, "broken.yml")); err == nil {
		t.Error("expected error for invalid yaml")
	}
}
