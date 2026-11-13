package workload

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/FacileStudio/gare/apps/api/internal/compose"
	"github.com/FacileStudio/gare/apps/api/internal/health"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
	"github.com/FacileStudio/gare/apps/api/internal/systemd"
	"github.com/FacileStudio/gare/apps/api/internal/testkit"
)

const suiteFixture = "testdata/gosvelteboilerplate-compose.yml"

func TestComposeServiceRoutesAnUnmodifiedSuiteFile(t *testing.T) {
	appDir := setupSuiteComposeApp(t)
	cfg := suiteComposeConfig()
	if err := (composeStack{}).Prepare(context.Background(), testApp(t, cfg, appDir), Create); err != nil {
		t.Fatalf("a suite compose file routed through a service must prepare: %v", err)
	}

	override := filepath.Join(appDir, compose.OverrideFileName)
	data, err := os.ReadFile(override)
	if err != nil {
		t.Fatalf("the override file was not written: %v", err)
	}
	if want := "services:\n    api:\n        ports:\n            - 127.0.0.1:8590:4000\n"; string(data) != want {
		t.Errorf("override: got\n%s\nwant\n%s", data, want)
	}
	unit := readUnit(t)
	podmanPath := systemd.ResolvePodmanPath()
	files := "compose -f docker-compose.yml -f " + override + " -p myapp "
	for _, want := range []string{
		"ExecStartPre=" + podmanPath + " network create --ignore dokploy-network\n",
		"ExecStart=" + podmanPath + " " + files + "up -d\n",
		"ExecStop=-" + podmanPath + " " + files + "down\n",
		"ExecStopPost=-" + podmanPath + " " + files + "down\n",
	} {
		if !strings.Contains(unit, want) {
			t.Errorf("expected the unit to contain %q, got:\n%s", want, unit)
		}
	}
}

func TestComposeServiceRefusesBadRoutingBeforeWriting(t *testing.T) {
	cases := map[string]func(cfg *storage.AppConfig){
		"unknown service":        func(cfg *storage.AppConfig) { cfg.Service = "web" },
		"missing container port": func(cfg *storage.AppConfig) { cfg.ContainerPort = 0 },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			appDir := setupSuiteComposeApp(t)
			cfg := suiteComposeConfig()
			mutate(cfg)
			if err := (composeStack{}).Prepare(context.Background(), testApp(t, cfg, appDir), Create); err == nil {
				t.Fatal("expected bad routing to be refused")
			}
			testkit.AssertGone(t, filepath.Join(appDir, compose.OverrideFileName))
			testkit.AssertGone(t, systemd.GetUnitPath("myapp"))
		})
	}
}

func TestComposeWithoutServiceKeepsThePublishedPortRule(t *testing.T) {
	appDir := setupSuiteComposeApp(t)
	cfg := suiteComposeConfig()
	cfg.Service = ""
	if err := (composeStack{}).Prepare(context.Background(), testApp(t, cfg, appDir), Create); err == nil {
		t.Error("a compose file that publishes no port must still be refused without a service")
	}

	appDir, _ = setupComposeApp(t, "8590:80")
	cfg = &storage.AppConfig{Name: "myapp", AppType: "compose", Port: 8590}
	if err := (composeStack{}).Prepare(context.Background(), testApp(t, cfg, appDir), Create); err != nil {
		t.Fatalf("preparing a compose app failed: %v", err)
	}
	testkit.AssertGone(t, filepath.Join(appDir, compose.OverrideFileName))
	unit := readUnit(t)
	if strings.Contains(unit, compose.OverrideFileName) || strings.Contains(unit, "ExecStartPre=") {
		t.Errorf("an app without a service must run the compose file alone, got:\n%s", unit)
	}
}

func TestComposeServiceSettlesAPortAndAProbeAtDeploy(t *testing.T) {
	appDir, repoDir := setupComposeApp(t, "1:1")
	writeRepoFile(t, repoDir, "docker-compose.yml", "services:\n  api:\n    expose:\n      - \"4000\"\n"+
		"    healthcheck:\n      test: [\"CMD\", \"wget\", \"-q\", \"http://localhost:4000/health\"]\n")
	cfg := &storage.AppConfig{Name: "myapp", AppType: "compose", ComposeRoute: storage.ComposeRoute{Service: "api"}, ContainerPort: 4000}
	if err := (composeStack{}).Settle(testApp(t, cfg, appDir), Deploy); err != nil {
		t.Fatalf("settling a service-routed app failed: %v", err)
	}
	if cfg.Port <= 0 {
		t.Fatalf("expected a gare-assigned port, got %d", cfg.Port)
	}
	want := []health.Probe{{Name: "api", Port: cfg.Port, Path: "/health"}}
	if got := cfg.HealthProbes(); !slices.Equal(got, want) {
		t.Errorf("probes: got %+v, want %+v", got, want)
	}
}

func setupSuiteComposeApp(t *testing.T) string {
	t.Helper()
	fixture, err := os.ReadFile(suiteFixture)
	if err != nil {
		t.Fatal(err)
	}
	appDir, repoDir := setupComposeApp(t, "1:1")
	writeRepoFile(t, repoDir, "docker-compose.yml", string(fixture))
	return appDir
}

func suiteComposeConfig() *storage.AppConfig {
	return &storage.AppConfig{Name: "myapp", AppType: "compose", Port: 8590, ComposeRoute: storage.ComposeRoute{Service: "api"}, ContainerPort: 4000}
}

func readUnit(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(systemd.GetUnitPath("myapp"))
	if err != nil {
		t.Fatalf("compose unit was not written: %v", err)
	}
	return string(data)
}

// writeSeededComposeUnit writes a minimal compose.yml into the app's checkout, then its compose
// unit, for a test that writes the unit without going through Prepare.
func writeSeededComposeUnit(ctx context.Context, app App) error {
	if err := os.MkdirAll(app.RepoDir(), 0755); err != nil {
		return err
	}
	composePath := filepath.Join(app.RepoDir(), "compose.yml")
	if err := os.WriteFile(composePath, []byte("services:\n  web:\n    image: nginx\n"), 0644); err != nil {
		return err
	}
	return writeComposeUnit(ctx, app, "compose.yml")
}
