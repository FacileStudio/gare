package compose

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/FacileStudio/gare/apps/api/internal/health"
)

func TestValidateService(t *testing.T) {
	repoDir := t.TempDir()
	writeComposeFile(t, repoDir, "compose.yml", "services:\n  api:\n    expose:\n      - \"4000\"\n  db:\n    image: postgres\n")
	composePath := filepath.Join(repoDir, "compose.yml")

	if err := ValidateService(composePath, "api", 4000); err != nil {
		t.Errorf("a declared service must validate: %v", err)
	}
	if err := ValidateService(composePath, "web", 4000); err == nil {
		t.Error("expected an error for a service the compose file does not declare")
	}
	for _, port := range []int{0, -1, 65536} {
		if err := ValidateService(composePath, "api", port); err == nil {
			t.Errorf("expected an error for container_port %d", port)
		}
	}
}

func TestWriteOverridePublishesOnlyTheRoutedPort(t *testing.T) {
	path := filepath.Join(t.TempDir(), OverrideFileName)
	if err := WriteOverride(path, Publish{Service: "api", HostPort: 8590, ContainerPort: 4000}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "services:\n    api:\n        ports:\n            - 127.0.0.1:8590:4000\n"
	if string(data) != want {
		t.Errorf("override: got\n%s\nwant\n%s", data, want)
	}
}

func TestExternalNetworks(t *testing.T) {
	repoDir := t.TempDir()
	content := "services:\n  api:\n    image: x\n" +
		"networks:\n" +
		"  dokploy-network:\n    external: true\n" +
		"  renamed:\n    external: true\n    name: shared-net\n" +
		"  legacy:\n    external:\n      name: old-net\n" +
		"  internal:\n    driver: bridge\n" +
		"  disabled:\n    external: false\n"
	writeComposeFile(t, repoDir, "compose.yml", content)

	got, err := ExternalNetworks(filepath.Join(repoDir, "compose.yml"))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"dokploy-network", "old-net", "shared-net"}
	if !slices.Equal(got, want) {
		t.Errorf("ExternalNetworks: got %v, want %v", got, want)
	}
}

func TestExternalNetworksRejectsAnUnsafeName(t *testing.T) {
	repoDir := t.TempDir()
	writeComposeFile(t, repoDir, "compose.yml", "networks:\n  bad:\n    external: true\n    name: \"a b\"\n")
	if _, err := ExternalNetworks(filepath.Join(repoDir, "compose.yml")); err == nil {
		t.Error("expected an error for a network name that would split a unit line")
	}
}

func TestPublishCountsAsPublished(t *testing.T) {
	repoDir := t.TempDir()
	content := "services:\n  api:\n    expose:\n      - \"4000\"\n" +
		"    healthcheck:\n      test: [\"CMD\", \"curl\", \"-f\", \"http://localhost:4000/health\"]\n"
	writeComposeFile(t, repoDir, "compose.yml", content)
	composePath := filepath.Join(repoDir, "compose.yml")
	publish := Publish{Service: "api", HostPort: 8590, ContainerPort: 4000}

	probes, err := HealthProbes(composePath, publish)
	if err != nil {
		t.Fatal(err)
	}
	want := []health.Probe{{Name: "api", Port: 8590, Path: "/health"}}
	if !slices.Equal(probes, want) {
		t.Errorf("HealthProbes with a publish: got %+v, want %+v", probes, want)
	}
	if loopback, err := LoopbackOnly(composePath, publish); err != nil || !loopback {
		t.Errorf("a gare publish is loopback only: got %v, err %v", loopback, err)
	}
	if probes, _ := HealthProbes(composePath); len(probes) != 0 {
		t.Errorf("without the publish the port is unreachable, got %+v", probes)
	}
}
