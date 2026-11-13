package workload

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/FacileStudio/gare/apps/api/internal/health"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
)

func TestResolveComposeFile(t *testing.T) {
	repoDir := t.TempDir()
	writeRepoFile(t, repoDir, "docker-compose.yml", "services:\n  web:\n    ports:\n      - \"8000:80\"\n")

	composeFile, err := resolveComposeFile(repoDir, "", 8000)
	if err != nil || composeFile != "docker-compose.yml" {
		t.Errorf("resolveComposeFile: got %q, err %v", composeFile, err)
	}
	if _, err := resolveComposeFile(repoDir, "", 8001); err == nil {
		t.Error("expected error when the port is not published by the compose file")
	}
	if _, err := resolveComposeFile(repoDir, "", 0); err == nil {
		t.Error("expected error when no port is configured")
	}
}

func TestProjectName(t *testing.T) {
	if got := ProjectName("My-App"); got != "my-app" {
		t.Errorf("ProjectName: got %q, want my-app", got)
	}
}

func TestComposeSettlesItsComposeFileAtCreate(t *testing.T) {
	appDir := t.TempDir()
	repoDir := storage.GetRepoDir(appDir)
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeRepoFile(t, repoDir, "compose.yml", "services:\n  web:\n    ports:\n      - \"8700:80\"\n")
	cfg := &storage.AppConfig{AppType: "compose", Port: 8700}
	if err := (composeStack{}).Settle(testApp(t, cfg, appDir), Create); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ComposeFile != "compose.yml" || cfg.AppType != "compose" {
		t.Errorf("unexpected settled config: %+v", cfg)
	}
	if cfg.ContainerPort != 0 {
		t.Errorf("compose workloads must not detect a container port, got %d", cfg.ContainerPort)
	}
}

func TestComposeSettleRefusesAMissingComposeFile(t *testing.T) {
	cfg := &storage.AppConfig{AppType: "compose", Port: 8700}
	if err := (composeStack{}).Settle(testApp(t, cfg, t.TempDir()), Create); err == nil {
		t.Error("expected error when no compose file exists")
	}
}

// composeWithHTTPHealthcheck is a compose file whose web service publishes 8700 and checks its
// health over HTTP, which is what lets a deploy derive a probe from it.
const composeWithHTTPHealthcheck = "services:\n  web:\n    ports:\n      - \"8700:80\"\n" +
	"    healthcheck:\n      test: [\"CMD\", \"curl\", \"-f\", \"http://localhost:80/health\"]\n"

// settleComposeDeploy settles a compose app at a deploy, from a checkout holding the given files.
func settleComposeDeploy(t *testing.T, files map[string]string) (*storage.AppConfig, *recordingReporter) {
	t.Helper()
	appDir := t.TempDir()
	repoDir := storage.GetRepoDir(appDir)
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		writeRepoFile(t, repoDir, name, content)
	}
	cfg := &storage.AppConfig{Name: "stack", AppType: "compose", Port: 8700}
	app := testApp(t, cfg, appDir)
	if err := (composeStack{}).Settle(app, Deploy); err != nil {
		t.Fatal(err)
	}
	return cfg, app.Report.(*recordingReporter)
}

// TestComposeDeployDerivesProbes pins that a deploy derives a compose app's probes from its compose
// file afresh, and says so.
func TestComposeDeployDerivesProbes(t *testing.T) {
	cfg, report := settleComposeDeploy(t, map[string]string{"compose.yml": composeWithHTTPHealthcheck})
	want := []health.Probe{{Name: "web", Port: 8700, Path: "/health"}}
	if got := cfg.HealthProbes(); !slices.Equal(got, want) {
		t.Errorf("derived probes = %+v, want %+v", got, want)
	}
	if len(report.lines) != 1 || !strings.HasPrefix(report.lines[0], "step: Derived 1 health probe(s) from compose.yml") {
		t.Errorf("expected the derivation to be reported, got %v", report.lines)
	}
}

// TestComposeDeployKeepsDeclaredProbes pins that a compose app whose gare.yml declares probes gets
// none derived from its compose file.
func TestComposeDeployKeepsDeclaredProbes(t *testing.T) {
	cfg, report := settleComposeDeploy(t, map[string]string{
		"compose.yml": composeWithHTTPHealthcheck,
		"gare.yml":    "healthchecks:\n  - port: 8700\n    path: /ready\n",
	})
	if got := cfg.HealthProbes(); len(got) != 1 || got[0].Path == "/health" {
		t.Errorf("a gare.yml with probes must not get derived ones, got %+v", got)
	}
	if len(report.lines) != 0 {
		t.Errorf("nothing is derived, so nothing is reported, got %v", report.lines)
	}
}

func TestProbeSummaryCapsLongLists(t *testing.T) {
	probes := make([]health.Probe, 0, 6)
	for i := range 6 {
		probes = append(probes, health.Probe{Name: fmt.Sprintf("svc%d", i), Port: 8100 + i})
	}
	summary := probeSummary(probes)
	if !strings.Contains(summary, "and 2 more") {
		t.Errorf("summary must cap the list, got %q", summary)
	}
	if strings.Contains(summary, "svc5") {
		t.Errorf("summary must omit the trailing labels, got %q", summary)
	}
	if short := probeSummary(probes[:2]); short != "svc0, svc1" {
		t.Errorf("short summary: got %q, want %q", short, "svc0, svc1")
	}
}

func TestContainerSettleDetectsTheExposedPort(t *testing.T) {
	appDir := t.TempDir()
	repoDir := storage.GetRepoDir(appDir)
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeRepoFile(t, repoDir, "Dockerfile", "FROM golang:alpine\nEXPOSE 8080\n")
	cfg := &storage.AppConfig{Name: "api", AppType: "container"}
	if err := (container{}).Settle(testApp(t, cfg, appDir), Create); err != nil {
		t.Fatal(err)
	}
	if cfg.ContainerPort != 8080 {
		t.Errorf("expected detected containerPort 8080 from Dockerfile, got %d", cfg.ContainerPort)
	}
	if cfg.Port != 0 {
		t.Errorf("a create allocates the host port itself, got %d", cfg.Port)
	}
	if err := (container{}).Settle(testApp(t, cfg, filepath.Join(appDir)), Deploy); err != nil {
		t.Fatal(err)
	}
	if cfg.Port == 0 {
		t.Error("a deploy of a container app stored without a port must allocate one")
	}
}
