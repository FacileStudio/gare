package engine

import (
	"slices"
	"testing"

	"github.com/FacileStudio/gare/apps/api/internal/manifest"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
)

// TestEnvChangesLandInTheWorkloadsStoreAndAreReported pins that an env edit reaches the store the
// app's workload reads, and that the sink hears about it before any restart, which is the order the
// operator has always seen it in.
func TestEnvChangesLandInTheWorkloadsStoreAndAreReported(t *testing.T) {
	store := storage.Open(t.TempDir())
	if err := store.Save(&storage.AppConfig{Name: "site", AppType: "static", Port: 8080}); err != nil {
		t.Fatal(err)
	}
	eng, sink := newTestEngine(store, Settings{})

	if _, err := eng.SetEnv(t.Context(), "site", map[string]string{"FOO": "bar"}); err != nil {
		t.Fatalf("SetEnv: %v", err)
	}
	if _, err := eng.LoadEnv(t.Context(), "site", map[string]string{"BAZ": "qux"}, "prod.env"); err != nil {
		t.Fatalf("LoadEnv: %v", err)
	}
	if _, err := eng.UnsetEnv(t.Context(), "site", []string{"FOO"}); err != nil {
		t.Fatalf("UnsetEnv: %v", err)
	}
	envs, err := eng.Env("site")
	if err != nil || len(envs) != 1 || envs["BAZ"] != "qux" {
		t.Fatalf("expected only BAZ=qux in the app env file, got %v err=%v", envs, err)
	}
	want := []string{
		"done: Updated 1 environment variable(s) in app env file",
		"done: Loaded 1 environment variable(s) from prod.env",
		"done: Removed 1 environment variable(s) from app env file",
	}
	if got := sink.recorded(); !slices.Equal(got, want) {
		t.Errorf("reported %v, want %v", got, want)
	}
}

// TestEngineRefusesAnUnknownApp pins the remedy every operation on a missing app reports.
func TestEngineRefusesAnUnknownApp(t *testing.T) {
	eng, sink := newTestEngine(storage.Open(t.TempDir()), Settings{})
	if _, err := eng.SetEnv(t.Context(), "ghost", map[string]string{"A": "b"}); err == nil {
		t.Fatal("expected an unknown app to be refused")
	}
	if _, err := eng.Status(t.Context(), "ghost"); err == nil {
		t.Fatal("expected an unknown app to be refused")
	}
	if got := sink.recorded(); len(got) != 0 {
		t.Errorf("a refused operation must report nothing, got %v", got)
	}
}

// TestWorkloadTypeChangeKeepsTheEnv pins that a container app an older gare left with its variables
// in the manifest keeps them when it becomes a compose app: the first locked operation, which a
// deploy runs before it resolves the new type, moves them into the env file every type reads.
func TestWorkloadTypeChangeKeepsTheEnv(t *testing.T) {
	store := storage.Open(t.TempDir())
	cfg := &storage.AppConfig{Name: "gare-envtype-test", AppType: "container", Port: 8080}
	if err := store.Save(cfg); err != nil {
		t.Fatal(err)
	}
	manifestPath := storage.GetManifestPath(store.AppDir(cfg.Name))
	if err := manifest.Generate(cfg.Name, 8080, 8080, manifestPath); err != nil {
		t.Fatal(err)
	}
	if err := manifest.RenderEnv(manifestPath, "", map[string]string{"DB_URL": "postgres://db"}); err != nil {
		t.Fatal(err)
	}
	eng, sink := newTestEngine(store, Settings{})
	if _, err := eng.AddTags(t.Context(), cfg.Name, []string{"api"}); err != nil {
		t.Fatalf("AddTags: %v", err)
	}
	if !sink.hasEvent("done: Moved 1 environment variable(s) from manifest.yaml") {
		t.Errorf("expected the migration to be reported, got %v", sink.recorded())
	}
	cfg.AppType = "compose"
	if err := store.Save(cfg); err != nil {
		t.Fatal(err)
	}
	envs, err := eng.Env(cfg.Name)
	if err != nil || envs["DB_URL"] != "postgres://db" {
		t.Fatalf("expected the env to survive the type change, got %v err=%v", envs, err)
	}
}
