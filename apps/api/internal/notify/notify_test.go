package notify

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/FacileStudio/gare/apps/api/internal/deploys"
)

// alert is what the fake Antenne decoded from one post.
type alert struct {
	Title   string         `json:"title"`
	Message string         `json:"message"`
	Event   string         `json:"event"`
	Source  string         `json:"source"`
	Data    map[string]any `json:"data"`
}

// fakeAntenne stands in for Antenne's generic webhook: it checks the path and the signature made
// with secret, answers 200, and hands every alert it decoded to the returned channel.
func fakeAntenne(t *testing.T, secret string) (string, chan alert) {
	t.Helper()
	got := make(chan alert, 4)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		if r.URL.Path != webhookPath || r.Header.Get(signatureHeader) != "sha256="+hex.EncodeToString(mac.Sum(nil)) {
			t.Errorf("post to %s with signature %q does not verify", r.URL.Path, r.Header.Get(signatureHeader))
		}
		var a alert
		if err := json.Unmarshal(body, &a); err != nil {
			t.Errorf("alert body is not JSON: %v", err)
		}
		got <- a
	}))
	t.Cleanup(server.Close)
	return server.URL, got
}

// configured returns the Notifier an environment pointing at url with secret builds.
func configured(t *testing.T, url, secret string) *Notifier {
	t.Helper()
	t.Setenv("ANTENNE_URL", url+"/")
	t.Setenv("ANTENNE_SECRET", secret)
	n := FromEnv()
	if n == nil {
		t.Fatal("ANTENNE_URL and ANTENNE_SECRET set must configure a notifier")
	}
	return n
}

// finished returns a finished deploy of app web with status, cause and the given rollback target.
func finished(status deploys.Status, cause string, to *string) deploys.Deploy {
	return deploys.Deploy{ID: "20261113-1", App: "web", Commit: "abc123", Trigger: deploys.TriggerWebhook,
		Status: status, Error: &cause, RolledBackTo: to}
}

// noWarn fails the test if a notifier reports a send failure.
func noWarn(t *testing.T) func(string) {
	return func(msg string) { t.Errorf("unexpected warning: %s", msg) }
}

func TestFailedDeployPostsOneErrorAlert(t *testing.T) {
	url, got := fakeAntenne(t, "s3cret")
	n := configured(t, url, "s3cret")
	n.Notify(finished(deploys.StatusFailed, "git sync failed", nil), noWarn(t))
	a := <-got
	want := map[string]any{"severity": "error", "app": "web", "deploy_id": "20261113-1",
		"commit": "abc123", "trigger": "webhook", "host": n.host}
	for key, value := range want {
		if a.Data[key] != value {
			t.Errorf("data[%s] = %v, want %v", key, a.Data[key], value)
		}
	}
	if a.Message != "git sync failed" || a.Event != "gare.deploy.failed" || a.Source != "gare" {
		t.Errorf("alert = %+v", a)
	}
	if len(got) != 0 {
		t.Error("a failed deploy must post exactly one alert")
	}
}

func TestRolledBackDeployPostsAWarning(t *testing.T) {
	url, got := fakeAntenne(t, "s3cret")
	to := "20261112-1"
	configured(t, url, "s3cret").Notify(finished(deploys.StatusRolledBack, "health check failed", &to), noWarn(t))
	a := <-got
	if a.Data["severity"] != "warning" || a.Data["rolled_back_to"] != to || a.Event != "gare.deploy.rolled_back" {
		t.Errorf("alert = %+v, want a warning naming the deploy it rolled back to", a)
	}
}

func TestSucceededDeployPostsNothing(t *testing.T) {
	url, got := fakeAntenne(t, "s3cret")
	configured(t, url, "s3cret").Notify(deploys.Deploy{ID: "1", App: "web", Status: deploys.StatusSucceeded}, noWarn(t))
	if len(got) != 0 {
		t.Error("a succeeded deploy must post nothing")
	}
}

func TestUnreachableAntenneOnlyWarns(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	server.Close()
	var warnings []string
	configured(t, server.URL, "s3cret").Notify(finished(deploys.StatusFailed, "boom", nil),
		func(msg string) { warnings = append(warnings, msg) })
	if len(warnings) != 1 || !strings.Contains(warnings[0], "Could not alert Antenne") {
		t.Errorf("warnings = %v, want one about Antenne", warnings)
	}
}
