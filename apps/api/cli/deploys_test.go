package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FacileStudio/gare/apps/api/internal/deploys"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
)

// seedDeploy saves an app and records one finished deploy of it with a log line, returning its id.
func seedDeploy(t *testing.T, name string) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	store := storage.Default()
	if err := store.Save(&storage.AppConfig{Name: name, Branch: "main"}); err != nil {
		t.Fatal(err)
	}
	r, err := deploys.Open(store.AppDir(name), name).Start(deploys.Request{Trigger: deploys.TriggerCLI, Commit: "abc1234"})
	if err != nil {
		t.Fatal(err)
	}
	r.Line("built the image")
	if err := r.Finish(nil); err != nil {
		t.Fatal(err)
	}
	return r.Record().ID
}

func TestDeploysListsAndPrintsALog(t *testing.T) {
	id := seedDeploy(t, "app")
	var list bytes.Buffer
	cmd := NewDeploysCmd()
	cmd.SetOut(&list)
	cmd.SetArgs([]string{"app"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(list.String(), id) || !strings.Contains(list.String(), "succeeded") {
		t.Errorf("the list must show the deploy, got:\n%s", list.String())
	}
	var log bytes.Buffer
	cmd = NewDeploysCmd()
	cmd.SetOut(&log)
	cmd.SetArgs([]string{"app", id, "--log"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if want := "built the image\ndeploy " + id + " succeeded\n"; log.String() != want {
		t.Errorf("log output = %q, want %q", log.String(), want)
	}
}
