package engine

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/FacileStudio/gare/apps/api/internal/deploys"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
)

// TestMain clears the Antenne settings an operator's shell may carry, so no engine test posts an
// alert to a real Antenne; the alert tests set their own.
func TestMain(m *testing.M) {
	for _, name := range []string{"ANTENNE_URL", "ANTENNE_INSTANCE", "ANTENNE_SECRET"} {
		os.Unsetenv(name)
	}
	os.Exit(m.Run())
}

// antenneAt points the engine's alerts at url for the rest of the test.
func antenneAt(t *testing.T, url string) {
	t.Helper()
	t.Setenv("ANTENNE_URL", url)
	t.Setenv("ANTENNE_SECRET", "s3cret")
}

// fakeAntenne records the data block of every alert it receives on the returned channel.
func fakeAntenne(t *testing.T) chan map[string]any {
	t.Helper()
	got := make(chan map[string]any, 4)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var alert struct {
			Data map[string]any `json:"data"`
		}
		if err := json.Unmarshal(body, &alert); err != nil {
			t.Errorf("alert body is not JSON: %v", err)
		}
		got <- alert.Data
	}))
	t.Cleanup(server.Close)
	antenneAt(t, server.URL)
	return got
}

// failingDeploy runs a deploy of an app with no checkout, which fails at its git sync.
func failingDeploy(t *testing.T, trigger deploys.Trigger) (*recordingSink, deploys.Deploy, error) {
	t.Helper()
	store := storage.Open(t.TempDir())
	if err := store.Save(&storage.AppConfig{Name: "app", Branch: "main"}); err != nil {
		t.Fatal(err)
	}
	eng, sink := newTestEngine(store, Settings{})
	_, err := eng.Deploy(t.Context(), "app", DeployOptions{Trigger: trigger})
	return sink, deployHistory(t, eng)[0], err
}

// nextAlert waits for the fake Antenne's next alert and fails the test when none arrives.
func nextAlert(t *testing.T, got chan map[string]any) map[string]any {
	t.Helper()
	select {
	case data := <-got:
		return data
	case <-time.After(10 * time.Second):
		t.Fatal("the failed deploy did not alert Antenne")
		return nil
	}
}

func TestFailedDeployAlertsAntenne(t *testing.T) {
	got := fakeAntenne(t)
	_, rec, err := failingDeploy(t, deploys.TriggerCLI)
	if err == nil {
		t.Fatal("a deploy without a checkout must fail")
	}
	data := nextAlert(t, got)
	if data["severity"] != "error" || data["app"] != "app" || data["deploy_id"] != rec.ID || data["trigger"] != "cli" {
		t.Errorf("alert data = %v, want an error naming deploy %s", data, rec.ID)
	}
	if len(got) != 0 {
		t.Error("a failed deploy must post exactly one alert")
	}
}

func TestDaemonDeployAlertsInTheBackground(t *testing.T) {
	got := fakeAntenne(t)
	if _, _, err := failingDeploy(t, deploys.TriggerWebhook); err == nil {
		t.Fatal("a deploy without a checkout must fail")
	}
	if data := nextAlert(t, got); data["trigger"] != "webhook" {
		t.Errorf("alert data = %v, want the webhook trigger", data)
	}
}

func TestUnreachableAntenneDoesNotFailTheDeploy(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	server.Close()
	antenneAt(t, server.URL)
	sink, rec, err := failingDeploy(t, deploys.TriggerCLI)
	if err == nil || strings.Contains(err.Error(), "Antenne") || rec.Status != deploys.StatusFailed {
		t.Errorf("deploy error = %v, record %+v; want the git failure alone", err, rec)
	}
	if !sink.hasEvent("warn: Could not alert Antenne about deploy " + rec.ID) {
		t.Errorf("events = %v, want a warning about Antenne", sink.recorded())
	}
}

func TestNoAlertWithoutAntenne(t *testing.T) {
	sink, _, _ := failingDeploy(t, deploys.TriggerCLI)
	for _, event := range sink.recorded() {
		if strings.Contains(event, "Antenne") {
			t.Errorf("an unconfigured host must say nothing about Antenne, got %q", event)
		}
	}
}
