package engine

import (
	"io"
	"strings"
	"testing"

	"github.com/FacileStudio/gare/apps/api/internal/deploys"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
)

// TestDeployRecordsItsOutcomeAndLog runs a deploy that fails at its git sync, the first step that
// touches the outside world, and checks the history: one record, finished as failed with the
// error, whose log holds the progress lines the sink saw and the output git wrote.
func TestDeployRecordsItsOutcomeAndLog(t *testing.T) {
	store := storage.Open(t.TempDir())
	if err := store.Save(&storage.AppConfig{Name: "app", Branch: "main"}); err != nil {
		t.Fatal(err)
	}
	eng, sink := newTestEngine(store, Settings{})
	_, deployErr := eng.Deploy(t.Context(), "app", DeployOptions{Trigger: deploys.TriggerWebhook})
	if deployErr == nil {
		t.Fatal("a deploy without a checkout must fail")
	}
	list := deployHistory(t, eng)
	if len(list) != 1 {
		t.Fatalf("history = %+v, want one record", list)
	}
	rec := list[0]
	if rec.Status != deploys.StatusFailed || rec.Trigger != deploys.TriggerWebhook || rec.Error == nil {
		t.Errorf("record = %+v, want a failed webhook deploy with its error", rec)
	}
	history, _ := eng.History("app")
	log := readLog(t, history, rec.ID)
	if !strings.Contains(log, "▸ Syncing app with its git remote...") {
		t.Errorf("the log must hold the sink's lines, got:\n%s", log)
	}
	if gitOutput := sink.stderr.String(); gitOutput == "" || !strings.Contains(log, strings.TrimSpace(gitOutput)) {
		t.Errorf("the log must hold git's output %q, got:\n%s", gitOutput, log)
	}
	if !strings.Contains(log, "✗ "+deployErr.Error()) {
		t.Errorf("the log must end with the error, got:\n%s", log)
	}
}

// TestQueuedDeployRunsUnderItsRecord checks a deploy queued ahead of time runs under the record the
// queue created instead of starting a second one.
func TestQueuedDeployRunsUnderItsRecord(t *testing.T) {
	store := storage.Open(t.TempDir())
	if err := store.Save(&storage.AppConfig{Name: "app", Branch: "main"}); err != nil {
		t.Fatal(err)
	}
	eng, _ := newTestEngine(store, Settings{})
	queued, err := eng.QueueDeploy("app", deploys.TriggerWebhook)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := eng.Deploy(t.Context(), "app", DeployOptions{Trigger: deploys.TriggerWebhook, QueuedID: queued.ID}); err == nil {
		t.Fatal("a deploy without a checkout must fail")
	}
	list := deployHistory(t, eng)
	if len(list) != 1 || list[0].ID != queued.ID || list[0].Status != deploys.StatusFailed {
		t.Errorf("history = %+v, want the queued record finished", list)
	}
}

// deployHistory returns the deploy records of the app named app.
func deployHistory(t *testing.T, eng *Engine) []deploys.Deploy {
	t.Helper()
	history, err := eng.History("app")
	if err != nil {
		t.Fatal(err)
	}
	list, err := history.List()
	if err != nil {
		t.Fatal(err)
	}
	return list
}

// readLog returns a deploy's whole log.
func readLog(t *testing.T, history deploys.History, id string) string {
	t.Helper()
	file, err := history.OpenLog(id)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
