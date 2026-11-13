package workload

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/FacileStudio/gare/apps/api/internal/manifest"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
)

const repoManifestWithEnv = "apiVersion: v1\nkind: Pod\nmetadata:\n  name: api\nspec:\n  containers:\n" +
	"  - name: api\n    image: localhost/api:latest\n    ports:\n    - containerPort: 3000\n" +
	"      hostPort: 8000\n    env:\n    - name: FROM_REPO\n      value: repo\n" +
	"    - name: CLASH\n      value: repo\n"

// containerApp returns a container app kept in a fresh directory with its checkout created.
func containerApp(t *testing.T) App {
	t.Helper()
	appDir := t.TempDir()
	if err := os.MkdirAll(storage.GetRepoDir(appDir), 0o755); err != nil {
		t.Fatal(err)
	}
	return testApp(t, &storage.AppConfig{Name: "api", Port: 8000, ContainerPort: 3000}, appDir)
}

func manifestEnv(t *testing.T, app App) map[string]string {
	t.Helper()
	envs, err := manifest.GetEnv(storage.GetManifestPath(app.Dir))
	if err != nil {
		t.Fatal(err)
	}
	return envs
}

func TestContainerDeployWritesTheEnvFileIntoTheManifest(t *testing.T) {
	app := containerApp(t)
	if err := storage.SetAppEnv(app.Dir, map[string]string{"SECRET": "ab #cd"}); err != nil {
		t.Fatal(err)
	}
	if err := syncManifest(app, ""); err != nil {
		t.Fatalf("syncManifest: %v", err)
	}
	if got := manifestEnv(t, app); len(got) != 1 || got["SECRET"] != "ab #cd" {
		t.Fatalf("expected the env file in the generated manifest, got %v", got)
	}
	if err := storage.UnsetAppEnv(app.Dir, []string{"SECRET"}); err != nil {
		t.Fatal(err)
	}
	if err := (container{}).RenderEnv(app); err != nil {
		t.Fatalf("RenderEnv: %v", err)
	}
	if err := syncManifest(app, ""); err != nil {
		t.Fatalf("second syncManifest: %v", err)
	}
	if got := manifestEnv(t, app); len(got) != 0 {
		t.Fatalf("expected a variable removed from the env file to leave the manifest, got %v", got)
	}
}

func TestRepoManifestEnvStaysAndTheEnvFileWins(t *testing.T) {
	app := containerApp(t)
	writeRepoFile(t, app.RepoDir(), "manifest.yaml", repoManifestWithEnv)
	if err := storage.SetAppEnv(app.Dir, map[string]string{"CLASH": "file", "OWN": "x"}); err != nil {
		t.Fatal(err)
	}
	if err := syncManifest(app, ""); err != nil {
		t.Fatalf("syncManifest: %v", err)
	}
	got := manifestEnv(t, app)
	if got["FROM_REPO"] != "repo" || got["CLASH"] != "file" || got["OWN"] != "x" || len(got) != 3 {
		t.Fatalf("expected the repo's entries kept and the env file to win, got %v", got)
	}
	if err := (container{}).RenderEnv(app); err != nil {
		t.Fatalf("RenderEnv: %v", err)
	}
	if again := manifestEnv(t, app); again["FROM_REPO"] != "repo" || again["CLASH"] != "file" {
		t.Fatalf("expected an env change to re-render on top of the repo's entries, got %v", again)
	}
}

func TestMigrateEnvMovesTheManifestEnvOnce(t *testing.T) {
	app := containerApp(t)
	writeRepoFile(t, app.RepoDir(), "manifest.yaml", repoManifestWithEnv)
	legacy := repoManifestWithEnv + "    - name: SET_BY_OPERATOR\n      value: s3cret\n"
	if err := os.WriteFile(storage.GetManifestPath(app.Dir), []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := MigrateEnv(app); err != nil {
			t.Fatalf("MigrateEnv: %v", err)
		}
	}
	envs, err := storage.GetAppEnv(app.Dir)
	if err != nil || len(envs) != 1 || envs["SET_BY_OPERATOR"] != "s3cret" {
		t.Fatalf("expected only the operator's variable in the env file, got %v err=%v", envs, err)
	}
	info, err := os.Stat(storage.GetAppEnvPath(app.Dir))
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("expected an owner-only env file, got %v err=%v", info, err)
	}
	if lines := app.Report.(*recordingReporter).lines; len(lines) != 1 {
		t.Fatalf("expected one migration line, got %v", lines)
	}
}

func TestDeployMigratesBeforeTheRepoManifestReplacesGaresCopy(t *testing.T) {
	app := containerApp(t)
	writeRepoFile(t, app.RepoDir(), "manifest.yaml", repoManifestWithEnv)
	legacy := repoManifestWithEnv + "    - name: TOKEN\n      value: t\n"
	if err := os.WriteFile(storage.GetManifestPath(app.Dir), []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := syncManifest(app, ""); err != nil {
		t.Fatalf("syncManifest: %v", err)
	}
	if got := manifestEnv(t, app); got["TOKEN"] != "t" || got["FROM_REPO"] != "repo" {
		t.Fatalf("expected the operator's variable to survive the repo sync, got %v", got)
	}
	if envs, _ := storage.GetAppEnv(app.Dir); envs["TOKEN"] != "t" {
		t.Fatalf("expected TOKEN in the env file, got %v", envs)
	}
}

func TestReadEnvShowsUnmigratedManifestEnvAndIgnoresItForOtherTypes(t *testing.T) {
	app := containerApp(t)
	if err := manifest.Generate("api", 3000, 8000, storage.GetManifestPath(app.Dir)); err != nil {
		t.Fatal(err)
	}
	if err := manifest.RenderEnv(storage.GetManifestPath(app.Dir), "", map[string]string{"OLD": "1"}); err != nil {
		t.Fatal(err)
	}
	if envs, err := ReadEnv(app); err != nil || envs["OLD"] != "1" {
		t.Fatalf("expected the unmigrated manifest variable, got %v err=%v", envs, err)
	}
	app.Config.AppType = "compose"
	if envs, err := ReadEnv(app); err != nil || len(envs) != 0 {
		t.Fatalf("a compose app must not read a stale manifest, got %v err=%v", envs, err)
	}
	if _, err := os.Stat(filepath.Join(app.Dir, "env")); !os.IsNotExist(err) {
		t.Fatalf("a read must not write the env file, got %v", err)
	}
}
