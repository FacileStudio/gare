package deploys

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/FacileStudio/gare/apps/api/internal/apikit/apitest"
	"github.com/FacileStudio/gare/apps/api/internal/deploys"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
	"github.com/go-chi/chi/v5"
)

// serve mounts the module over a fixture holding one static app named demo, with no checkout, so
// a deploy of it fails at its git sync.
func serve(t *testing.T) (*httptest.Server, *apitest.Fixture) {
	t.Helper()
	f := apitest.New(t)
	f.AddApp(t, &storage.AppConfig{Name: "demo", AppType: "static", Branch: "main"})
	srv := apitest.Serve(t, func(r chi.Router) { RegisterRoutes(r, NewService(f.Deps), apitest.Pass) })
	return srv, f
}

// history is demo's deploy history.
func history(t *testing.T, f *apitest.Fixture) deploys.History {
	t.Helper()
	h, err := f.Deps.Engine.History("demo")
	if err != nil {
		t.Fatal(err)
	}
	return h
}

// readStream reads a whole event stream, with lastID sent as Last-Event-ID when set.
func readStream(t *testing.T, srv *httptest.Server, path, lastID string) (string, string) {
	t.Helper()
	request, _ := http.NewRequest(http.MethodGet, srv.URL+path, nil)
	if lastID != "" {
		request.Header.Set("Last-Event-ID", lastID)
	}
	response, err := srv.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	return response.Header.Get("Content-Type"), string(body)
}

// TestDeployAnswersItsQueuedRecord proves POST deploy answers 202 with an api record at once, and
// that the history and the record routes find it once it ran.
func TestDeployAnswersItsQueuedRecord(t *testing.T) {
	srv, f := serve(t)
	status, body := apitest.Do(t, srv, http.MethodPost, "/api/apps/demo/deploy", "")
	var rec Deploy
	if err := json.Unmarshal([]byte(body), &rec); err != nil || status != http.StatusAccepted || rec.Trigger != deploys.TriggerAPI {
		t.Fatalf("POST deploy = %d %s", status, body)
	}
	if _, err := history(t, f).Follow(t.Context(), rec.ID, 0, func(int, string) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if _, body := apitest.Do(t, srv, http.MethodGet, "/api/apps/demo/deploys", ""); !strings.Contains(body, `"status":"failed"`) {
		t.Fatalf("history = %s, want the failed deploy", body)
	}
	if status, _ := apitest.Do(t, srv, http.MethodGet, "/api/apps/demo/deploys/"+rec.ID, ""); status != http.StatusOK {
		t.Fatalf("GET the deploy = %d", status)
	}
	for _, path := range []string{"/api/apps/demo/deploys/nope", "/api/apps/nope/deploys", "/api/apps/nope/deploy"} {
		method := map[bool]string{true: http.MethodPost, false: http.MethodGet}[strings.HasSuffix(path, "/deploy")]
		if status, _ := apitest.Do(t, srv, method, path, ""); status != http.StatusNotFound {
			t.Errorf("%s %s = %d, want 404", method, path, status)
		}
	}
}

// TestLogReplaysAndResumes proves a finished deploy's log is replayed with line numbers as ids and
// ends with done, and that Last-Event-ID resumes after the line it names.
func TestLogReplaysAndResumes(t *testing.T) {
	srv, f := serve(t)
	rec, err := history(t, f).Start(deploys.Request{Trigger: deploys.TriggerCLI})
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range []string{"one", "two", "three"} {
		rec.Line(line)
	}
	if err := rec.Finish(nil); err != nil {
		t.Fatal(err)
	}
	path := "/api/apps/demo/deploys/" + rec.Record().ID + "/log"
	kind, body := readStream(t, srv, path, "")
	want := "id: 1\ndata: one\n\nid: 2\ndata: two\n\nid: 3\ndata: three\n\nevent: done\ndata: {"
	if kind != "text/event-stream" || !strings.HasPrefix(body, want) || !strings.Contains(body, `"status":"succeeded"`) {
		t.Fatalf("log = %q %q", kind, body)
	}
	if _, body := readStream(t, srv, path, "2"); !strings.HasPrefix(body, "id: 3\ndata: three\n\nevent: done\n") {
		t.Fatalf("resumed log = %q", body)
	}
}

// TestLogFollowsARunningDeploy proves lines written after the stream opened arrive, then done.
func TestLogFollowsARunningDeploy(t *testing.T) {
	srv, f := serve(t)
	rec, err := history(t, f).Start(deploys.Request{Trigger: deploys.TriggerAPI})
	if err != nil {
		t.Fatal(err)
	}
	rec.Line("before")
	go func() {
		time.Sleep(300 * time.Millisecond)
		rec.Line("after")
		if err := rec.Finish(nil); err != nil {
			t.Error(err)
		}
	}()
	_, body := readStream(t, srv, "/api/apps/demo/deploys/"+rec.Record().ID+"/log", "")
	if !strings.HasPrefix(body, "id: 1\ndata: before\n\nid: 2\ndata: after\n\nevent: done\n") {
		t.Fatalf("followed log = %q", body)
	}
}

// TestRollbackOfAStaticAppIsRefused proves the engine's refusal reaches the client as a 412 that
// says why, and records nothing.
func TestRollbackOfAStaticAppIsRefused(t *testing.T) {
	srv, f := serve(t)
	status, body := apitest.Do(t, srv, http.MethodPost, "/api/apps/demo/rollback", "{}")
	if status != http.StatusPreconditionFailed || !strings.Contains(body, "container") {
		t.Fatalf("rollback = %d %s", status, body)
	}
	if list, _ := history(t, f).List(); len(list) != 0 {
		t.Fatalf("a refused rollback recorded %+v", list)
	}
}
