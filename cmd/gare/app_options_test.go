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
		domain:        "app.local",
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
	yamlContent := "type: static\nstatic_dir: build\nbuild_cmd: npm run build\n"
	if err := os.WriteFile(filepath.Join(repoDir, "gare.yaml"), []byte(yamlContent), 0644); err != nil {
		t.Fatal(err)
	}
	opts := appCreateOptions{
		repo:   "https://example.com/repo.git",
		domain: "static.local",
	}
	resolved, err := resolveAppOptions(tmpDir, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.appType != "static" {
		t.Errorf("expected static appType from gare.yaml, got %s", resolved.appType)
	}
	if resolved.staticDir != "build" {
		t.Errorf("expected build staticDir from gare.yaml, got %s", resolved.staticDir)
	}
	if resolved.buildCmd != "npm run build" {
		t.Errorf("expected buildCmd from gare.yaml, got %s", resolved.buildCmd)
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
		domain:        "custom.local",
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

func TestStaticSnippetGeneration(t *testing.T) {
	snippet, err := caddy.GenerateStaticSnippet("static.test", "/var/www/dist")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(snippet, "file_server") {
		t.Errorf("expected file_server in snippet, got: %s", snippet)
	}
	if !strings.Contains(snippet, "try_files") {
		t.Errorf("expected try_files in snippet, got: %s", snippet)
	}
}

func TestStaticConfigStorage(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "static-app")
	cfg := &storage.AppConfig{
		Name:      "static-app",
		Domain:    "static.test",
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
		Domain:  "static.local",
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
