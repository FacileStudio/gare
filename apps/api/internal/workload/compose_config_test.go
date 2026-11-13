package workload

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/FacileStudio/gare/apps/api/internal/compose"
	"github.com/FacileStudio/gare/apps/api/internal/podman"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
	"gopkg.in/yaml.v3"
)

type mergedPort struct {
	HostIP    string `yaml:"host_ip"`
	Target    int    `yaml:"target"`
	Published any    `yaml:"published"`
}

type mergedConfig struct {
	Services map[string]struct {
		Ports []mergedPort `yaml:"ports"`
	} `yaml:"services"`
}

func TestComposeProviderMergesTheOverride(t *testing.T) {
	skipWithoutComposeProvider(t)
	appDir := setupSuiteComposeApp(t)
	cfg := suiteComposeConfig()
	if err := (composeStack{}).Prepare(context.Background(), testApp(t, cfg, appDir), Create); err != nil {
		t.Fatal(err)
	}
	override := filepath.Join(appDir, compose.OverrideFileName)
	cmd := exec.Command("podman", "compose", "-f", "docker-compose.yml", "-f", override, "-p", "myapp", "config")
	cmd.Dir = storage.GetRepoDir(appDir)
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("podman compose config failed: %v", err)
	}
	var merged mergedConfig
	if err := yaml.Unmarshal(output, &merged); err != nil {
		t.Fatalf("unreadable merged config: %v\n%s", err, output)
	}
	ports := merged.Services["api"].Ports
	if len(ports) != 1 || ports[0].HostIP != "127.0.0.1" || ports[0].Target != 4000 ||
		fmt.Sprint(ports[0].Published) != "8590" {
		t.Errorf("the merged config must publish 127.0.0.1:8590:4000, got %+v\n%s", ports, output)
	}
}

func skipWithoutComposeProvider(t *testing.T) {
	t.Helper()
	err := podman.CheckComposeProvider(context.Background())
	if err == nil {
		return
	}
	if os.Getenv("GARE_REQUIRE_PODMAN") == "1" {
		t.Fatalf("podman compose is unavailable: %v", err)
	}
	t.Skipf("podman compose is unavailable here: %v", err)
}
