package engine

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/FacileStudio/gare/apps/api/internal/ingress"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
	"github.com/FacileStudio/gare/apps/api/internal/testkit"
	"github.com/FacileStudio/gare/apps/api/internal/workload"
)

type artifactCase struct {
	workload   storage.WorkloadType
	volumes    bool
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
		"compose keeps volumes by default": {
			workload: storage.WorkloadCompose,
			cfg:      &storage.AppConfig{Name: "myapp", AppType: "compose", ComposeRoute: storage.ComposeRoute{ComposeFile: "compose.yml"}},
			wantPodman: []string{
				"compose -f compose.yml -p myapp down",
				"ps -a --filter label=com.docker.compose.project=myapp --format {{.ID}}",
			},
		},
		"compose with --volumes": {
			workload: storage.WorkloadCompose,
			volumes:  true,
			cfg:      &storage.AppConfig{Name: "myapp", AppType: "compose", ComposeRoute: storage.ComposeRoute{ComposeFile: "compose.yml"}},
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
	store := storage.Open(t.TempDir())
	appDir := store.AppDir("myapp")
	logPath := testkit.StubPodman(t)
	confDir := t.TempDir()
	t.Setenv("GARE_CADDY_CONF_DIR", confDir)
	t.Setenv("GARE_CADDYFILE", filepath.Join(confDir, "Caddyfile"))
	fakeCaddy(t)
	seedSnippet(t, confDir, "myapp")
	seedComposeRepo(t, appDir, tc.workload)

	plan := destroyPlan{name: "myapp", cfg: tc.cfg, volumes: tc.volumes}
	if tc.cfg != nil {
		plan.workload, _ = workload.For(tc.cfg)
	}
	eng, _ := newTestEngine(store, Settings{})
	eng.removeArtifacts(context.Background(), plan)

	if got := testkit.RecordedCalls(t, logPath); !slices.Equal(got, tc.wantPodman) {
		t.Errorf("podman invocations = %v, want %v", got, tc.wantPodman)
	}
	testkit.AssertGone(t, ingress.GetSnippetPath(confDir, "myapp"))
	testkit.AssertGone(t, appDir)
}

func seedSnippet(t *testing.T, confDir, name string) {
	t.Helper()
	if err := os.WriteFile(ingress.GetSnippetPath(confDir, name), []byte(name+".example {\n}\n"), 0644); err != nil {
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
