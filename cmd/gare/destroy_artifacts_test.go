package main

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/FacileStudio/gare/internal/caddy"
	"github.com/FacileStudio/gare/internal/storage"
)

type artifactCase struct {
	workload   storage.WorkloadType
	cfg        *storage.AppConfig
	wantPodman []string
}

// TestRemoveArtifactsTearsDownOnlyTheWorkloadItCanName pins what a destroy runs for each workload
// type, and that an unknown type takes only the artifacts that do not depend on it — the unit, the
// ingress snippet and the storage — instead of tearing down a container workload's image on a guess.
func TestRemoveArtifactsTearsDownOnlyTheWorkloadItCanName(t *testing.T) {
	for name, tc := range artifactCases() {
		t.Run(name, func(t *testing.T) {
			assertArtifactTeardown(t, tc)
		})
	}
}

// artifactCases returns each workload type with the podman commands its teardown is allowed to run.
func artifactCases() map[string]artifactCase {
	return map[string]artifactCase{
		"container": {
			workload:   storage.WorkloadContainer,
			cfg:        &storage.AppConfig{Name: "myapp", AppType: "container"},
			wantPodman: []string{"rmi -f localhost/myapp:latest"},
		},
		"compose": {
			workload: storage.WorkloadCompose,
			cfg:      &storage.AppConfig{Name: "myapp", AppType: "compose", ComposeFile: "compose.yml"},
			wantPodman: []string{
				"compose -f compose.yml -p myapp down -v",
				"ps -a --filter label=com.docker.compose.project=myapp --format {{.ID}}",
			},
		},
		"unknown": {
			workload:   storage.WorkloadUnknown,
			cfg:        nil,
			wantPodman: nil,
		},
	}
}

func assertArtifactTeardown(t *testing.T, tc artifactCase) {
	t.Helper()
	appDir := t.TempDir()
	logPath := stubPodman(t)
	confDir := t.TempDir()
	t.Setenv("GARE_CADDY_CONF_DIR", confDir)
	seedSnippet(t, confDir, "myapp")
	seedComposeRepo(t, appDir, tc.workload)

	removeArtifacts(context.Background(), "myapp", appDir, tc.workload, tc.cfg)

	if got := podmanCalls(t, logPath); !slices.Equal(got, tc.wantPodman) {
		t.Errorf("podman invocations = %v, want %v", got, tc.wantPodman)
	}
	assertGone(t, caddy.GetSnippetPath(confDir, "myapp"))
	assertGone(t, appDir)
}

// stubPodman puts a recording `podman` first on PATH and returns the file it logs each invocation
// to, so the commands a teardown runs can be asserted without depending on a real podman.
func stubPodman(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	logPath := filepath.Join(dir, "podman.log")
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"$GARE_TEST_PODMAN_LOG\"\nexit 0\n"
	if err := os.WriteFile(filepath.Join(dir, "podman"), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("GARE_TEST_PODMAN_LOG", logPath)
	return logPath
}

func podmanCalls(t *testing.T, logPath string) []string {
	t.Helper()
	data, err := os.ReadFile(logPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatalf("failed to read the podman call log: %v", err)
	}
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "\n")
}

func seedSnippet(t *testing.T, confDir, name string) {
	t.Helper()
	if err := os.WriteFile(caddy.GetSnippetPath(confDir, name), []byte(name+".example {\n}\n"), 0644); err != nil {
		t.Fatal(err)
	}
}

func seedComposeRepo(t *testing.T, appDir string, workload storage.WorkloadType) {
	t.Helper()
	if workload != storage.WorkloadCompose {
		return
	}
	repoDir := storage.GetRepoDir(appDir)
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "compose.yml"), []byte("services: {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
}
