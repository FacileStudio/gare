package cli

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FacileStudio/gare/apps/api/internal/storage"
)

func TestResolveComposeWorkload(t *testing.T) {
	repoDir := t.TempDir()
	writeRepoComposeFile(t, repoDir, "docker-compose.yml", "services:\n  web:\n    ports:\n      - \"8000:80\"\n")

	composeFile, err := resolveComposeWorkload(repoDir, "", 8000)
	if err != nil || composeFile != "docker-compose.yml" {
		t.Errorf("resolveComposeWorkload: got %q, err %v", composeFile, err)
	}
	if _, err := resolveComposeWorkload(repoDir, "", 8001); err == nil {
		t.Error("expected error when the port is not published by the compose file")
	}
	if _, err := resolveComposeWorkload(repoDir, "", 0); err == nil {
		t.Error("expected error when no port is configured")
	}
}

func TestComposeProjectName(t *testing.T) {
	if got := composeProjectName("My-App"); got != "my-app" {
		t.Errorf("composeProjectName: got %q, want my-app", got)
	}
}

func TestNormalizeCreateConfigCompose(t *testing.T) {
	repoDir := t.TempDir()
	writeRepoComposeFile(t, repoDir, "compose.yml", "services:\n  web:\n    ports:\n      - \"8700:80\"\n")
	resolved := &storage.AppConfig{AppType: "compose", Port: 8700}
	if err := normalizeCreateConfig(repoDir, resolved); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.ComposeFile != "compose.yml" || resolved.AppType != "compose" {
		t.Errorf("unexpected resolved options: %+v", resolved)
	}
	if resolved.ContainerPort != 0 {
		t.Errorf("compose workloads must not detect a container port, got %d", resolved.ContainerPort)
	}
}

func TestNormalizeCreateConfigMissingComposeFile(t *testing.T) {
	cfg := &storage.AppConfig{AppType: "compose", Port: 8700}
	if err := normalizeCreateConfig(t.TempDir(), cfg); err == nil {
		t.Error("expected error when no compose file exists")
	}
}

func writeRepoComposeFile(t *testing.T, repoDir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(repoDir, name), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestResolveDeployConfigTakesGareFileChanges(t *testing.T) {
	repoDir := t.TempDir()
	writeRepoComposeFile(t, repoDir, "gare.yml", "build_cmd: make prod\nstatic_dir: build\n")
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
	repoDir := t.TempDir()
	writeRepoComposeFile(t, repoDir, "gare.yml", fmt.Sprintf("port: %d\n", busy))
	cfg := &storage.AppConfig{Name: "app", Port: 8000}
	err = resolveDeployConfig(storage.Open(t.TempDir()), repoDir, cfg)
	if err == nil || !strings.Contains(err.Error(), "not available") {
		t.Fatalf("a gare.yml port that is taken must be refused, got %v", err)
	}
	if cfg.Port != 8000 {
		t.Errorf("a refused port must leave the stored one, got %d", cfg.Port)
	}
}
