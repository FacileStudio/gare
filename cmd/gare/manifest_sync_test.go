package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FacileStudio/gare/internal/storage"
)

func TestSyncManifestFromRepo(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")
	repoDir := filepath.Join(appDir, "repo")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatal(err)
	}
	repoManifest := filepath.Join(repoDir, "manifest.yaml")
	content := "apiVersion: v1\nkind: Pod\nmetadata:\n  name: custom\n"
	if err := os.WriteFile(repoManifest, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg := &storage.AppConfig{Name: "custom", Port: 8080, ContainerPort: 3000}
	if err := syncManifest("custom", appDir, repoDir, cfg); err != nil {
		t.Fatalf("syncManifest failed: %v", err)
	}
	appManifest := storage.GetManifestPath(appDir)
	data, err := os.ReadFile(appManifest)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "name: custom") {
		t.Errorf("expected synced manifest, got: %s", string(data))
	}
}

func TestSyncManifestGenerate(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")
	repoDir := filepath.Join(appDir, "repo")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatal(err)
	}
	cfg := &storage.AppConfig{Name: "autogen", Port: 8000, ContainerPort: 8080}
	if err := syncManifest("autogen", appDir, repoDir, cfg); err != nil {
		t.Fatalf("initial syncManifest generation failed: %v", err)
	}
	appManifest := storage.GetManifestPath(appDir)
	data, err := os.ReadFile(appManifest)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "containerPort: 8080") || !strings.Contains(string(data), "hostPort: 8000") {
		t.Errorf("expected initial ports, got: %s", string(data))
	}
}

func TestSyncManifestMutate(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")
	repoDir := filepath.Join(appDir, "repo")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatal(err)
	}
	cfg := &storage.AppConfig{Name: "autogen", Port: 8000, ContainerPort: 8080}
	if err := syncManifest("autogen", appDir, repoDir, cfg); err != nil {
		t.Fatal(err)
	}
	cfg.Port = 9000
	cfg.ContainerPort = 4000
	if err := syncManifest("autogen", appDir, repoDir, cfg); err != nil {
		t.Fatalf("mutation syncManifest failed: %v", err)
	}
	appManifest := storage.GetManifestPath(appDir)
	data, err := os.ReadFile(appManifest)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "containerPort: 4000") || !strings.Contains(string(data), "hostPort: 9000") {
		t.Errorf("expected mutated ports, got: %s", string(data))
	}
}

func TestSyncManifestPinsLocalImagePullPolicy(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")
	repoDir := filepath.Join(appDir, "repo")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatal(err)
	}
	existing := "apiVersion: v1\nkind: Pod\nmetadata:\n  name: vitrine\n" +
		"spec:\n  containers:\n  - name: vitrine\n    image: localhost/vitrine:latest\n" +
		"    ports:\n    - containerPort: 3012\n      hostPort: 8000\n"
	if err := os.WriteFile(storage.GetManifestPath(appDir), []byte(existing), 0644); err != nil {
		t.Fatal(err)
	}
	cfg := &storage.AppConfig{Name: "vitrine", Port: 8000, ContainerPort: 3012}
	if err := syncManifest("vitrine", appDir, repoDir, cfg); err != nil {
		t.Fatalf("syncManifest failed: %v", err)
	}
	data, err := os.ReadFile(storage.GetManifestPath(appDir))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "imagePullPolicy: Never") {
		t.Errorf("expected the local image pull policy to be pinned, got: %s", string(data))
	}
}

func TestResolveAppOptionsStaticInference(t *testing.T) {
	tmpDir := t.TempDir()
	opts := appCreateOptions{
		repo:      "https://example.com/repo.git",
		staticDir: "dist",
	}
	resolved, err := resolveAppOptions(tmpDir, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.appType != "static" {
		t.Errorf("expected inferred static appType, got %q", resolved.appType)
	}
}
