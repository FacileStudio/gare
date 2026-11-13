package engine

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/FacileStudio/gare/apps/api/internal/deploys"
	"github.com/FacileStudio/gare/apps/api/internal/manifest"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
	"github.com/FacileStudio/gare/apps/api/internal/testkit"
)

// stubFlakySystemctl puts a systemctl first on PATH that accepts every verb and reports the unit
// failed the first time it is asked for its state and active after that: a new build that does not
// come up, then the restored one that does.
func stubFlakySystemctl(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	script := "#!/bin/sh\n[ \"$2\" = show ] || exit 0\n" +
		"if [ -e \"$0.seen\" ]; then echo ActiveState=active; exit 0; fi\n" +
		": > \"$0.seen\"\nprintf 'ActiveState=failed\\nSubState=failed\\nResult=exit-code\\n'\n"
	if err := os.WriteFile(filepath.Join(dir, "systemctl"), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// seedSucceededDeploy records a successful deploy of commit and returns its id.
func seedSucceededDeploy(t *testing.T, store *storage.Store, name, commit string) string {
	t.Helper()
	r, err := deploys.Open(store.AppDir(name), name).Start(deploys.Request{Trigger: deploys.TriggerCLI, Commit: commit})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Finish(nil); err != nil {
		t.Fatal(err)
	}
	return r.Record().ID
}

// TestFailedHealthCheckRollsBackToThePreviousImage restarts a new build that never becomes active
// and checks the deploy goes back to the previous successful one: the manifest names its tag,
// :latest moves to it, and the failed deploy is recorded rolled-back to it.
func TestFailedHealthCheckRollsBackToThePreviousImage(t *testing.T) {
	podmanLog := testkit.StubPodman(t)
	stubFlakySystemctl(t)
	store, cfg := containerAppFixture(t)
	previous := seedSucceededDeploy(t, store, "web", "aaaaaaa")
	eng, _ := newTestEngine(store, Settings{})
	run, rec, err := eng.startRecord("web", deploys.Request{Trigger: deploys.TriggerCLI, Commit: "bbbbbbb"}, "")
	if err != nil {
		t.Fatal(err)
	}
	deployErr := run.restartAndVerify(t.Context(), cfg, "bbbbbbb")
	eng.finishRecord("web", rec, deployErr)

	if rolled, ok := asRolledBack(deployErr); !ok || rolled.To != previous {
		t.Fatalf("deploy error = %v, want a rollback to %s", deployErr, previous)
	}
	assertManifestImage(t, store, "localhost/web:aaaaaaa")
	if calls := testkit.RecordedCalls(t, podmanLog); !slices.Contains(calls, "tag localhost/web:aaaaaaa localhost/web:latest") {
		t.Errorf("a rollback must move :latest to the restored image, podman calls %v", calls)
	}
	got, err := deploys.Open(store.AppDir("web"), "web").Get(rec.Record().ID)
	if err != nil || got.Status != deploys.StatusRolledBack || got.RolledBackTo == nil || *got.RolledBackTo != previous {
		t.Errorf("record = %+v, %v; want rolled-back to %s", got, err, previous)
	}
}

// containerAppFixture stores a container app named web with a generated manifest, served on the
// port of a test server that answers, so its health probe passes once the unit reports active.
func containerAppFixture(t *testing.T) (*storage.Store, *storage.AppConfig) {
	t.Helper()
	store := storage.Open(t.TempDir())
	app := newTestServer(t, map[string]int{"/": 200})
	cfg := &storage.AppConfig{Name: "web", Branch: "main", AppType: "container", Port: app.port}
	if err := store.Save(cfg); err != nil {
		t.Fatal(err)
	}
	if err := manifest.Generate("web", 3000, app.port, storage.GetManifestPath(store.AppDir("web"))); err != nil {
		t.Fatal(err)
	}
	return store, cfg
}

// assertManifestImage fails unless the app's manifest runs image.
func assertManifestImage(t *testing.T, store *storage.Store, image string) {
	t.Helper()
	data, err := os.ReadFile(storage.GetManifestPath(store.AppDir("web")))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "image: "+image) {
		t.Errorf("manifest must run %s, got:\n%s", image, data)
	}
}

func TestRollbackRefusesAStaticApp(t *testing.T) {
	store := storage.Open(t.TempDir())
	if err := store.Save(&storage.AppConfig{Name: "site", AppType: "static", StaticDir: "dist"}); err != nil {
		t.Fatal(err)
	}
	eng, _ := newTestEngine(store, Settings{})
	_, err := eng.Rollback(t.Context(), "site", RollbackOptions{Trigger: deploys.TriggerCLI})
	if err == nil || !strings.Contains(err.Error(), "only supports container workloads") {
		t.Errorf("rolling back a static app must be refused clearly, got %v", err)
	}
	if eng.CanRollback(t.Context(), "site") {
		t.Error("a static app must never report it can roll back")
	}
}

func TestImageTagRetentionKeepsFiveCommits(t *testing.T) {
	var list []deploys.Deploy
	for _, commit := range []string{"7777777", "6666666", "6666666", "5555555", "", "4444444", "3333333", "2222222", "1111111"} {
		list = append(list, deploys.Deploy{Commit: commit})
	}
	keep := keepTags(list)
	if want := []string{"7777777", "6666666", "5555555", "4444444", "3333333"}; !slices.Equal(keep, want) {
		t.Fatalf("keepTags = %v, want %v", keep, want)
	}
	tags := []string{"latest", "7777777", "3333333", "2222222", "1111111", "stable"}
	if stale := staleTags(tags, keep); !slices.Equal(stale, []string{"2222222", "1111111"}) {
		t.Errorf("staleTags = %v, want the two oldest commits only", stale)
	}
}
