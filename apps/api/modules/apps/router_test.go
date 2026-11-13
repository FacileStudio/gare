package apps

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/FacileStudio/gare/apps/api/internal/apikit/apitest"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
	"github.com/go-chi/chi/v5"
)

// staticFixture is a repository a static app can be created from.
var staticFixture = map[string]string{"index.html": "<h1>hi</h1>", "gare.yml": "type: static\nstatic_dir: .\n"}

// serve mounts the module over a fixture holding one static app named demo.
func serve(t *testing.T) (*httptest.Server, *apitest.Fixture) {
	t.Helper()
	f := apitest.New(t)
	f.AddApp(t, &storage.AppConfig{Name: "demo", AppType: "static", Branch: "main", Port: 8123, StaticDir: "."})
	srv := apitest.Serve(t, func(r chi.Router) { RegisterRoutes(r, NewService(f.Deps), apitest.Pass) })
	return srv, f
}

// decodeApp reads an App answer.
func decodeApp(t *testing.T, body string) App {
	t.Helper()
	var app App
	if err := json.Unmarshal([]byte(body), &app); err != nil {
		t.Fatalf("decode %q: %v", body, err)
	}
	return app
}

// TestListAndReadAnApp proves the list and the read answer the contract's shape, with empty lists
// as [] and no deploy as null, and an unknown app as a 404 envelope.
func TestListAndReadAnApp(t *testing.T) {
	srv, _ := serve(t)
	status, body := apitest.Do(t, srv, http.MethodGet, "/api/apps", "")
	if status != http.StatusOK || !strings.Contains(body, `"domains":[]`) || !strings.Contains(body, `"last_deploy":null`) {
		t.Fatalf("GET /api/apps = %d %s", status, body)
	}
	status, body = apitest.Do(t, srv, http.MethodGet, "/api/apps/demo", "")
	if app := decodeApp(t, body); status != http.StatusOK || app.Status != "inactive" || app.Type != "static" {
		t.Fatalf("GET /api/apps/demo = %d %+v", status, app)
	}
	status, body = apitest.Do(t, srv, http.MethodGet, "/api/apps/nope", "")
	if status != http.StatusNotFound || !strings.Contains(body, `"message":"there is no app named \"nope\""`) {
		t.Fatalf("GET /api/apps/nope = %d %s", status, body)
	}
}

// TestCreateRefusesUnsafeRepositories proves a repository git could read as an option, or one of
// an unexpected scheme, never reaches the engine.
func TestCreateRefusesUnsafeRepositories(t *testing.T) {
	srv, _ := serve(t)
	for _, repo := range []string{"", "-oProxyCommand=evil", "file:///etc", "http://example.test/x.git", "relative/path"} {
		body, _ := json.Marshal(CreateRequest{Name: "x", RepoURL: repo})
		if status, answer := apitest.Do(t, srv, http.MethodPost, "/api/apps", string(body)); status != http.StatusBadRequest {
			t.Errorf("repo %q = %d %s, want 400", repo, status, answer)
		}
	}
	body := `{"name":"x","repo_url":"https://example.test/x.git","branch":"--upload-pack=evil"}`
	if status, _ := apitest.Do(t, srv, http.MethodPost, "/api/apps", body); status != http.StatusBadRequest {
		t.Errorf("a branch starting with '-' = %d, want 400", status)
	}
}

// TestCreateClonesThroughTheEngine creates a static app from a local repository, then refuses a
// second app of the same name.
func TestCreateClonesThroughTheEngine(t *testing.T) {
	srv, _ := serve(t)
	body, _ := json.Marshal(CreateRequest{Name: "site", RepoURL: apitest.GitRepo(t, staticFixture)})
	status, answer := apitest.Do(t, srv, http.MethodPost, "/api/apps", string(body))
	if app := decodeApp(t, answer); status != http.StatusCreated || app.Name != "site" || app.Type != "static" || app.Port == 0 {
		t.Fatalf("POST /api/apps = %d %s", status, answer)
	}
	if status, answer := apitest.Do(t, srv, http.MethodPost, "/api/apps", string(body)); status != http.StatusConflict {
		t.Fatalf("second create = %d %s, want 409", status, answer)
	}
}

// TestDestroyNeedsTheNameTypedBack proves a destroy without the right confirm changes nothing.
func TestDestroyNeedsTheNameTypedBack(t *testing.T) {
	srv, _ := serve(t)
	if status, _ := apitest.Do(t, srv, http.MethodDelete, "/api/apps/demo", `{"confirm":"dem"}`); status != http.StatusBadRequest {
		t.Fatalf("wrong confirm = %d, want 400", status)
	}
	if status, _ := apitest.Do(t, srv, http.MethodGet, "/api/apps/demo", ""); status != http.StatusOK {
		t.Fatalf("the app must survive a refused destroy, got %d", status)
	}
	if status, body := apitest.Do(t, srv, http.MethodDelete, "/api/apps/demo", `{"confirm":"demo"}`); status != http.StatusNoContent {
		t.Fatalf("destroy = %d %s, want 204", status, body)
	}
	if status, _ := apitest.Do(t, srv, http.MethodGet, "/api/apps/demo", ""); status != http.StatusNotFound {
		t.Fatalf("a destroyed app = %d, want 404", status)
	}
}

// TestStartAndStopAnswerTheApp proves the lifecycle routes answer the app as systemd then reports.
func TestStartAndStopAnswerTheApp(t *testing.T) {
	srv, _ := serve(t)
	status, body := apitest.Do(t, srv, http.MethodPost, "/api/apps/demo/start", "")
	if app := decodeApp(t, body); status != http.StatusOK || app.Status != "active" {
		t.Fatalf("start = %d %s", status, body)
	}
	status, body = apitest.Do(t, srv, http.MethodPost, "/api/apps/demo/stop", "")
	if app := decodeApp(t, body); status != http.StatusOK || app.Status != "inactive" {
		t.Fatalf("stop = %d %s", status, body)
	}
}

// TestABusyAppAnswersConflictQuickly holds the app's lock and proves a change answers 409 after the
// short wait instead of hanging.
func TestABusyAppAnswersConflictQuickly(t *testing.T) {
	srv, f := serve(t)
	unlock, err := f.Deps.Store.Lock(context.Background(), "demo")
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	began := time.Now()
	status, body := apitest.Do(t, srv, http.MethodPost, "/api/apps/demo/restart", "")
	if status != http.StatusConflict || !strings.Contains(body, "another operation on demo is running") {
		t.Fatalf("restart of a busy app = %d %s", status, body)
	}
	if waited := time.Since(began); waited > 5*time.Second {
		t.Fatalf("the busy answer took %v", waited)
	}
}

// TestTagsAreReplaced proves PUT sets the exact set, normalised.
func TestTagsAreReplaced(t *testing.T) {
	srv, _ := serve(t)
	_, body := apitest.Do(t, srv, http.MethodPut, "/api/apps/demo/tags", `{"tags":["Web","prod"]}`)
	if app := decodeApp(t, body); strings.Join(app.Tags, ",") != "web,prod" {
		t.Fatalf("tags = %v, want web,prod", app.Tags)
	}
	_, body = apitest.Do(t, srv, http.MethodPut, "/api/apps/demo/tags", `{"tags":["prod"]}`)
	if app := decodeApp(t, body); strings.Join(app.Tags, ",") != "prod" {
		t.Fatalf("tags = %v, want prod", app.Tags)
	}
	if status, _ := apitest.Do(t, srv, http.MethodPut, "/api/apps/demo/tags", `{"tags":["no way"]}`); status != http.StatusBadRequest {
		t.Fatalf("an invalid tag = %d, want 400", status)
	}
}

// TestLogsStreamTheJournal proves the journal arrives as one data event per line, and that a count
// over the cap is refused before the stream opens.
func TestLogsStreamTheJournal(t *testing.T) {
	srv, _ := serve(t)
	response, err := srv.Client().Get(srv.URL + "/api/apps/demo/logs?lines=50")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	if response.Header.Get("Content-Type") != "text/event-stream" || string(body) != "data: first line\n\ndata: second line\n\n" {
		t.Fatalf("logs = %q %q", response.Header.Get("Content-Type"), body)
	}
	if status, _ := apitest.Do(t, srv, http.MethodGet, "/api/apps/demo/logs?lines=5000", ""); status != http.StatusBadRequest {
		t.Fatalf("lines=5000 = %d, want 400", status)
	}
}
