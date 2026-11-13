package daemon

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"

	"github.com/FacileStudio/gare/apps/api/internal/apikit/apitest"
	"github.com/FacileStudio/gare/apps/api/internal/deploys"
	"github.com/FacileStudio/gare/apps/api/modules/apps"
)

// send makes one cookie request with a JSON body and the CSRF header the
// client sends on every mutation.
func (h *harness) send(method, path, body string) (int, string) {
	h.t.Helper()
	request, err := http.NewRequest(method, h.base+path, strings.NewReader(body))
	if err != nil {
		h.t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Facile-CSRF", "1")
	response, err := h.client.Do(request)
	if err != nil {
		h.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer response.Body.Close()
	data, _ := io.ReadAll(response.Body)
	return response.StatusCode, string(data)
}

// TestOperatorRoutesNeedASession proves a JSON route and both stream routes
// answer 401 JSON without a session, never a redirect to sign-in.
func TestOperatorRoutesNeedASession(t *testing.T) {
	h := newHarness(t, "", "ops@example.test")
	for _, path := range []string{"/api/apps", "/api/apps/demo/logs", "/api/apps/demo/deploys/x/log"} {
		response := h.do(http.MethodGet, h.base+path, nil)
		kind := response.Header.Get("Content-Type")
		if response.StatusCode != http.StatusUnauthorized || !strings.HasPrefix(kind, "application/json") {
			t.Errorf("GET %s = %d %s, want a 401 JSON answer", path, response.StatusCode, kind)
		}
	}
}

// TestAStaticAppDeploysEndToEnd boots the whole router, signs in, creates a
// static app from a local repository, deploys it through the API and follows
// its log over the event stream until done says it succeeded. systemd, podman
// and Caddy are stubs; the health probe is answered by a listener on the
// app's port.
func TestAStaticAppDeploysEndToEnd(t *testing.T) {
	h := newHarness(t, "", "ops@example.test")
	h.signIn("ops@example.test")
	repo := apitest.GitRepo(t, map[string]string{"index.html": "<h1>hi</h1>", "gare.yml": "type: static\nstatic_dir: .\n"})
	status, body := h.send(http.MethodPost, "/api/apps", fmt.Sprintf(`{"name":"site","repo_url":%q}`, repo))
	var app apps.App
	if status != http.StatusCreated || json.Unmarshal([]byte(body), &app) != nil {
		t.Fatalf("create = %d %s", status, body)
	}
	answerOn(t, app.Port)
	status, body = h.send(http.MethodPost, "/api/apps/site/deploy", "")
	var queued deploys.Deploy
	if status != http.StatusAccepted || json.Unmarshal([]byte(body), &queued) != nil || queued.Trigger != deploys.TriggerAPI {
		t.Fatalf("deploy = %d %s", status, body)
	}
	done := followLog(t, h, "/api/apps/site/deploys/"+queued.ID+"/log")
	if done.Status != deploys.StatusSucceeded || done.ID != queued.ID || done.Commit == "" {
		t.Fatalf("done = %+v, want the queued deploy succeeded", done)
	}
	_, body = h.send(http.MethodGet, "/api/apps/site", "")
	if json.Unmarshal([]byte(body), &app) != nil || app.LastDeploy == nil || app.LastDeploy.ID != queued.ID || app.Status != "active" {
		t.Fatalf("app after deploy = %s", body)
	}
}

// answerOn serves any request on 127.0.0.1:port for the rest of the test, as
// the static container would.
func answerOn(t *testing.T, port int) {
	t.Helper()
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatalf("listen on the app's port: %v", err)
	}
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})}
	go srv.Serve(listener)
	t.Cleanup(func() { srv.Close() })
}

// followLog reads a deploy's event stream to its end, checks every line came
// with its number as id, and returns the Deploy the done event carries.
func followLog(t *testing.T, h *harness, path string) deploys.Deploy {
	t.Helper()
	response := h.do(http.MethodGet, h.base+path, nil)
	if kind := response.Header.Get("Content-Type"); response.StatusCode != http.StatusOK || kind != "text/event-stream" {
		t.Fatalf("GET %s = %d %s", path, response.StatusCode, kind)
	}
	scanner := bufio.NewScanner(response.Body)
	lines, event := 0, ""
	for scanner.Scan() {
		text := scanner.Text()
		switch {
		case text == fmt.Sprintf("id: %d", lines+1):
			lines++
		case text == "event: done":
			event = "done"
		case event == "done" && strings.HasPrefix(text, "data: "):
			var final deploys.Deploy
			if err := json.Unmarshal([]byte(strings.TrimPrefix(text, "data: ")), &final); err != nil {
				t.Fatalf("done data %q: %v", text, err)
			}
			t.Logf("followed %d log lines to done", lines)
			return final
		}
	}
	t.Fatalf("the stream ended without done after %d lines", lines)
	return deploys.Deploy{}
}
