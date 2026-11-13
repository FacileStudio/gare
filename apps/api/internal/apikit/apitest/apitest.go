// Package apitest builds what the HTTP module tests share: a real engine over a temp store, with
// systemd, podman and Caddy stubbed, the webhook queue it deploys through, and a router that mounts
// one module under /api with authentication passed through. It is imported by tests only.
package apitest

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FacileStudio/gare/apps/api/internal/apikit"
	"github.com/FacileStudio/gare/apps/api/internal/engine"
	"github.com/FacileStudio/gare/apps/api/internal/server"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
	"github.com/FacileStudio/gare/apps/api/internal/testkit"
	"github.com/go-chi/chi/v5"
)

// Quiet is a sink that drops what the engine reports; the deploy log keeps it anyway.
type Quiet struct{}

// Event drops a progress line.
func (Quiet) Event(string, engine.Kind, string) {}

// Output discards subprocess output.
func (Quiet) Output(string) (io.Writer, io.Writer) { return io.Discard, io.Discard }

// Fixture is one test's daemon half: the dependencies a module is built over and the queue.
type Fixture struct {
	Deps   apikit.Deps
	Intake *server.Server
}

// New returns a fixture whose engine runs against stubs only, so no test touches the host's units,
// containers or Caddy.
func New(t *testing.T) *Fixture {
	t.Helper()
	testkit.StubSystemd(t)
	testkit.StubPodman(t)
	testkit.StubCommand(t, "caddy", "#!/bin/sh\nexit 0\n")
	t.Setenv("GARE_CADDY_CONF_DIR", t.TempDir())
	t.Setenv("GARE_CADDYFILE", filepath.Join(t.TempDir(), "Caddyfile"))
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	store := storage.Open(filepath.Join(t.TempDir(), "apps"))
	eng := engine.New(store, Quiet{}, engine.Settings{})
	intake := server.New(server.Config{Secret: "test"})
	intake.UseDeployer(eng)
	t.Cleanup(func() {
		if err := intake.Shutdown(context.Background()); err != nil {
			t.Errorf("shutdown: %v", err)
		}
	})
	return &Fixture{Deps: apikit.Deps{Engine: eng, Store: store, Queue: intake, AllowLocalRepos: true}, Intake: intake}
}

// AddApp records an app in the store without cloning anything.
func (f *Fixture) AddApp(t *testing.T, cfg *storage.AppConfig) {
	t.Helper()
	if err := f.Deps.Store.Save(cfg); err != nil {
		t.Fatal(err)
	}
}

// Pass is an authentication middleware that lets every request through.
func Pass(next http.Handler) http.Handler { return next }

// Serve mounts register under /api on a test listener.
func Serve(t *testing.T, register func(chi.Router)) *httptest.Server {
	t.Helper()
	router := chi.NewRouter()
	router.Route("/api", register)
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)
	return srv
}

// GitRepo commits files to a new repository on branch main and returns its absolute path, the
// local fixture a create clones from.
func GitRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	git := "git init -q -b main && git add -A && " +
		"git -c user.name=test -c user.email=test@example.test commit -q -m fixture"
	if output, err := exec.Command("sh", "-c", "cd "+dir+" && "+git).CombinedOutput(); err != nil {
		t.Fatalf("git fixture: %v: %s", err, output)
	}
	return dir
}

// Do sends one request with a JSON body, when body is not empty, and returns the status and body.
func Do(t *testing.T, srv *httptest.Server, method, path, body string) (int, string) {
	t.Helper()
	request, err := http.NewRequest(method, srv.URL+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := srv.Client().Do(request)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return response.StatusCode, string(data)
}
