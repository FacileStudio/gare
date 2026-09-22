package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FacileStudio/gare/internal/caddy"
	"github.com/FacileStudio/gare/internal/storage"
)

func TestResolveAppOptionsWithFlags(t *testing.T) {
	tmpDir := t.TempDir()
	opts := appCreateOptions{
		repo:          "https://example.com/repo.git",
		appType:       "static",
		staticDir:     "dist",
		containerfile: "apps/api/Dockerfile",
		contextDir:    "apps/api",
		buildCmd:      "bun run build",
	}
	resolved, err := resolveAppOptions(tmpDir, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.appType != "static" {
		t.Errorf("expected static appType, got %s", resolved.appType)
	}
	if resolved.staticDir != "dist" {
		t.Errorf("expected staticDir dist, got %s", resolved.staticDir)
	}
	if resolved.containerfile != "apps/api/Dockerfile" {
		t.Errorf("expected containerfile apps/api/Dockerfile, got %s", resolved.containerfile)
	}
	if resolved.contextDir != "apps/api" {
		t.Errorf("expected contextDir apps/api, got %s", resolved.contextDir)
	}
	if resolved.buildCmd != "bun run build" {
		t.Errorf("expected buildCmd bun run build, got %s", resolved.buildCmd)
	}
}

func TestResolveAppOptionsWithGareFile(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := storage.GetRepoDir(tmpDir)
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatal(err)
	}
	yamlContent := "type: static\nstatic_dir: build\nbuild_cmd: npm run build\nport: 8080\ndomain: static.local\n"
	if err := os.WriteFile(filepath.Join(repoDir, "gare.yaml"), []byte(yamlContent), 0644); err != nil {
		t.Fatal(err)
	}
	opts := appCreateOptions{
		repo: "https://example.com/repo.git",
	}
	resolved, err := resolveAppOptions(tmpDir, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.appType != "static" || resolved.staticDir != "build" {
		t.Errorf("unexpected static opts: %+v", resolved)
	}
	if resolved.port != 8080 {
		t.Errorf("expected port 8080, got %d", resolved.port)
	}
}

func TestResolveAppOptionsPrecedence(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := storage.GetRepoDir(tmpDir)
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatal(err)
	}
	yamlContent := "type: container\ncontainerfile: Dockerfile.base\nbuild_cmd: make\n"
	if err := os.WriteFile(filepath.Join(repoDir, "gare.yml"), []byte(yamlContent), 0644); err != nil {
		t.Fatal(err)
	}
	opts := appCreateOptions{
		repo:          "https://example.com/repo.git",
		containerfile: "Custom.Dockerfile",
		buildCmd:      "make prod",
	}
	resolved, err := resolveAppOptions(tmpDir, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.containerfile != "Custom.Dockerfile" {
		t.Errorf("expected flag override Custom.Dockerfile, got %s", resolved.containerfile)
	}
	if resolved.buildCmd != "make prod" {
		t.Errorf("expected flag override make prod, got %s", resolved.buildCmd)
	}
}

func TestStaticIngressProxiesToTheContainerPort(t *testing.T) {
	confDir := t.TempDir()
	t.Setenv("GARE_CADDY_CONF_DIR", confDir)
	t.Setenv("GARE_CADDYFILE", filepath.Join(confDir, "Caddyfile"))
	cfg := &storage.AppConfig{
		Name:      "static-app",
		Domains:   []string{"static.test"},
		Port:      8080,
		AppType:   "static",
		StaticDir: "public",
	}

	if err := syncAppIngress(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	content, err := os.ReadFile(caddy.GetSnippetPath(confDir, "static-app"))
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

func TestStaticConfigStorage(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "static-app")
	cfg := &storage.AppConfig{
		Name:      "static-app",
		Domains:   []string{"static.test"},
		AppType:   "static",
		StaticDir: "public",
		BuildCmd:  "npm run build",
	}
	if err := storage.SaveConfig(appDir, cfg); err != nil {
		t.Fatal(err)
	}
	loaded, err := storage.LoadConfig(appDir)
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.IsStatic() || loaded.StaticDir != "public" {
		t.Errorf("expected static app with public staticDir, got: %+v", loaded)
	}
}

func TestListCmdStaticTable(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	appDir := filepath.Join(tmpDir, ".local", "share", "gare", "apps", "staticapp")
	cfg := &storage.AppConfig{
		Name:    "staticapp",
		Port:    0,
		Domains: []string{"static.local"},
		AppType: "static",
	}
	if err := storage.SaveConfig(appDir, cfg); err != nil {
		t.Fatal(err)
	}
	cmd := NewListCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "staticapp") {
		t.Errorf("expected output to contain staticapp, got %q", output)
	}
	if !strings.Contains(output, "static") {
		t.Errorf("expected output to display static status, got %q", output)
	}
}

func TestResolveAppOptionsContainerPortFlag(t *testing.T) {
	tmpDir := t.TempDir()
	opts := appCreateOptions{
		repo:          "https://example.com/repo.git",
		containerPort: 4000,
	}
	resolved, err := resolveAppOptions(tmpDir, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.containerPort != 4000 {
		t.Errorf("expected containerPort 4000, got %d", resolved.containerPort)
	}
}

func TestResolveAppOptionsContainerPortExpose(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := storage.GetRepoDir(tmpDir)
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatal(err)
	}
	dfContent := "FROM golang:alpine\nEXPOSE 8080\n"
	if err := os.WriteFile(filepath.Join(repoDir, "Dockerfile"), []byte(dfContent), 0644); err != nil {
		t.Fatal(err)
	}
	opts := appCreateOptions{repo: "https://example.com/repo.git"}
	resolved, err := resolveAppOptions(tmpDir, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.containerPort != 8080 {
		t.Errorf("expected detected containerPort 8080 from Dockerfile, got %d", resolved.containerPort)
	}
}
