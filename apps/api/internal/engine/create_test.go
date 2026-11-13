package engine

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FacileStudio/gare/apps/api/internal/appconfig"
	"github.com/FacileStudio/gare/apps/api/internal/ingress"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
)

const testRepo = "https://example.com/repo.git"

// resolveTestCreate resolves a create request against the checkout under appDir.
func resolveTestCreate(t *testing.T, appDir string, flags appconfig.Flags) (*storage.AppConfig, error) {
	t.Helper()
	eng, _ := newTestEngine(storage.Open(t.TempDir()), Settings{})
	cfg, _, err := eng.resolveCreate(appDir, CreateRequest{Name: "app", Repo: testRepo, Flags: flags})
	return cfg, err
}

// writeGareFile writes a gare.yml into the checkout under appDir.
func writeGareFile(t *testing.T, appDir, name, content string) string {
	t.Helper()
	repoDir := storage.GetRepoDir(appDir)
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, name), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return repoDir
}

func TestResolveCreateWithFlags(t *testing.T) {
	resolved, err := resolveTestCreate(t, t.TempDir(), appconfig.Flags{
		Type: "static", StaticDir: "dist", Containerfile: "apps/api/Dockerfile", Context: "apps/api", BuildCmd: "bun run build",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.AppType != "static" || resolved.StaticDir != "dist" {
		t.Errorf("expected a static app serving dist, got %+v", resolved)
	}
	if resolved.Containerfile != "apps/api/Dockerfile" || resolved.ContextDir != "apps/api" {
		t.Errorf("expected the build paths from the flags, got %+v", resolved)
	}
	if resolved.BuildCmd != "bun run build" {
		t.Errorf("expected buildCmd bun run build, got %s", resolved.BuildCmd)
	}
	if resolved.Name != "app" || resolved.RepoURL != testRepo {
		t.Errorf("expected the request's name and repository, got %+v", resolved)
	}
}

func TestResolveCreateWithGareFile(t *testing.T) {
	appDir := t.TempDir()
	writeGareFile(t, appDir, "gare.yaml", "type: static\nstatic_dir: build\nbuild_cmd: npm run build\nport: 8080\ndomain: static.local\n")
	resolved, err := resolveTestCreate(t, appDir, appconfig.Flags{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.AppType != "static" || resolved.StaticDir != "build" || resolved.Port != 8080 {
		t.Errorf("unexpected static opts: %+v", resolved)
	}
}

func TestResolveCreatePrecedence(t *testing.T) {
	appDir := t.TempDir()
	writeGareFile(t, appDir, "gare.yml", "type: container\ncontainerfile: Dockerfile.base\nbuild_cmd: make\n")
	resolved, err := resolveTestCreate(t, appDir, appconfig.Flags{Containerfile: "Custom.Dockerfile", BuildCmd: "make prod"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.Containerfile != "Custom.Dockerfile" || resolved.BuildCmd != "make prod" {
		t.Errorf("expected the flags to override gare.yml, got %+v", resolved)
	}
}

func TestResolveCreateContainerPorts(t *testing.T) {
	resolved, err := resolveTestCreate(t, t.TempDir(), appconfig.Flags{ContainerPort: 4000})
	if err != nil || resolved.ContainerPort != 4000 {
		t.Errorf("expected containerPort 4000 from the flag, got %+v err=%v", resolved, err)
	}
	appDir := t.TempDir()
	writeGareFile(t, appDir, "Dockerfile", "FROM golang:alpine\nEXPOSE 8080\n")
	resolved, err = resolveTestCreate(t, appDir, appconfig.Flags{})
	if err != nil || resolved.ContainerPort != 8080 {
		t.Errorf("expected detected containerPort 8080 from Dockerfile, got %+v err=%v", resolved, err)
	}
}

func TestResolveCreateStaticInference(t *testing.T) {
	resolved, err := resolveTestCreate(t, t.TempDir(), appconfig.Flags{StaticDir: "dist"})
	if err != nil || resolved.AppType != "static" {
		t.Errorf("expected inferred static appType, got %+v err=%v", resolved, err)
	}
}

func TestDeriveName(t *testing.T) {
	cases := map[string]string{
		"https://github.com/saravenpi/test-gare":     "test-gare",
		"https://github.com/saravenpi/test-gare.git": "test-gare",
		"git@github.com:saravenpi/my-service.git":    "my-service",
		"https://gitlab.com/group/subgroup/api-v2/":  "api-v2",
		"custom-name": "custom-name",
		"":            "",
	}
	for url, want := range cases {
		if got := DeriveName(url); got != want {
			t.Errorf("DeriveName(%q) = %q, want %q", url, got, want)
		}
	}
}

func TestStaticIngressProxiesToTheContainerPort(t *testing.T) {
	confDir := t.TempDir()
	t.Setenv("GARE_CADDY_CONF_DIR", confDir)
	t.Setenv("GARE_CADDYFILE", filepath.Join(confDir, "Caddyfile"))
	cfg := &storage.AppConfig{Name: "static-app", Domains: []string{"static.test"}, Port: 8080, AppType: "static", StaticDir: "public"}

	if err := ingress.WriteSnippet(confDir, cfg.Name, cfg.Domains, cfg.Port); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	content, err := os.ReadFile(ingress.GetSnippetPath(confDir, "static-app"))
	if err != nil {
		t.Fatalf("failed to read the snippet: %v", err)
	}
	snippet := string(content)
	if !strings.Contains(snippet, "reverse_proxy localhost:8080") {
		t.Errorf("static ingress must proxy to the container port, got: %s", snippet)
	}
	for _, forbidden := range []string{"file_server", "root *"} {
		if strings.Contains(snippet, forbidden) {
			t.Errorf("the host must not serve static files itself (%q), got: %s", forbidden, snippet)
		}
	}
}

func TestRepoPathsOutsideTheRepositoryAreRejected(t *testing.T) {
	flags := map[string]appconfig.Flags{
		"--static":        {StaticDir: "../x"},
		"--context":       {Context: "../x"},
		"--containerfile": {Containerfile: "../x"},
	}
	for key, set := range flags {
		err := validateCreate(CreateRequest{Name: "app", Repo: testRepo, Flags: set})
		if err == nil || !strings.Contains(err.Error(), key) {
			t.Errorf("flag %s=../x: expected a rejection naming it, got %v", key, err)
		}
	}
	for _, key := range []string{"static_dir", "context", "containerfile"} {
		appDir := t.TempDir()
		repoDir := writeGareFile(t, appDir, "gare.yml", key+": ../x\n")
		_, createErr := resolveTestCreate(t, appDir, appconfig.Flags{})
		deployErr := resolveDeployConfig(storage.Open(t.TempDir()), repoDir, &storage.AppConfig{Name: "app"})
		for _, err := range []error{createErr, deployErr} {
			if err == nil || !strings.Contains(err.Error(), key) || !strings.Contains(err.Error(), "../x") {
				t.Errorf("gare.yml %s: ../x: expected a rejection naming key and value, got %v", key, err)
			}
		}
	}
}

func TestResolveDeployConfigTakesGareFileChanges(t *testing.T) {
	repoDir := writeGareFile(t, t.TempDir(), "gare.yml", "build_cmd: make prod\nstatic_dir: build\n")
	cfg := &storage.AppConfig{Name: "app", AppType: "static", StaticDir: "dist", BuildCmd: "make", Port: 8000}
	if err := resolveDeployConfig(storage.Open(t.TempDir()), repoDir, cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.BuildCmd != "make prod" || cfg.StaticDir != "build" {
		t.Errorf("a deploy must apply what gare.yml changed since create, got %+v", cfg)
	}
}

func TestResolveDeployConfigRefusesABusyGareFilePort(t *testing.T) {
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	busy := ln.Addr().(*net.TCPAddr).Port
	repoDir := writeGareFile(t, t.TempDir(), "gare.yml", fmt.Sprintf("port: %d\n", busy))
	cfg := &storage.AppConfig{Name: "app", Port: 8000}
	err = resolveDeployConfig(storage.Open(t.TempDir()), repoDir, cfg)
	if err == nil || !strings.Contains(err.Error(), "not available") {
		t.Fatalf("a gare.yml port that is taken must be refused, got %v", err)
	}
	if cfg.Port != 8000 {
		t.Errorf("a refused port must leave the stored one, got %d", cfg.Port)
	}
}
