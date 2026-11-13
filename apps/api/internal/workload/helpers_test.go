package workload

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/FacileStudio/gare/apps/api/internal/storage"
	"github.com/FacileStudio/gare/apps/api/internal/systemd"
)

// recordingReporter keeps every line an adapter reports, prefixed with its kind, so a test can
// assert what an operator would have been told.
type recordingReporter struct {
	lines []string
}

func (r *recordingReporter) Step(msg string)    { r.lines = append(r.lines, "step: "+msg) }
func (r *recordingReporter) Done(msg string)    { r.lines = append(r.lines, "done: "+msg) }
func (r *recordingReporter) Warn(msg string)    { r.lines = append(r.lines, "warn: "+msg) }
func (r *recordingReporter) Verbose(msg string) { r.lines = append(r.lines, "verbose: "+msg) }

// testApp returns an app named after cfg, kept in dir, reporting to a fresh recording reporter.
func testApp(t *testing.T, cfg *storage.AppConfig, dir string) App {
	t.Helper()
	return App{
		Config: cfg,
		Dir:    dir,
		Store:  storage.Open(t.TempDir()),
		Report: &recordingReporter{},
		Stdout: io.Discard,
		Stderr: io.Discard,
	}
}

// skipWithoutPodmanWorkloads lets a machine without a usable podman skip these tests, while CI
// fails loudly: the point of them is to prove the generated units actually run, so a silent skip
// there would hide exactly the regression they exist to catch. CI sets GARE_REQUIRE_PODMAN.
func skipWithoutPodmanWorkloads(t *testing.T) {
	t.Helper()
	err := systemd.CheckPodmanWorkloads(context.Background())
	if err == nil {
		return
	}
	if os.Getenv("GARE_REQUIRE_PODMAN") == "1" {
		t.Fatalf("podman cannot run the generated units: %v", err)
	}
	t.Skipf("podman cannot run the generated units here: %v", err)
}

func assertUnitContent(t *testing.T, unitPath, want string) {
	t.Helper()
	data, err := os.ReadFile(unitPath)
	if err != nil {
		t.Fatalf("expected %s to be left in place, got %v", unitPath, err)
	}
	if string(data) != want {
		t.Errorf("expected %s to keep its own definition, got:\n%s", unitPath, string(data))
	}
}

// seedLegacyQuadletSources leaves the Quadlet sources a gare that predates unit synthesis wrote, so
// a test can prove a workload write retires them.
func seedLegacyQuadletSources(t *testing.T, name string) {
	t.Helper()
	for _, path := range systemd.LegacyQuadletSources(name) {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("[Kube]\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func writeRepoFile(t *testing.T, repoDir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(repoDir, name), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
