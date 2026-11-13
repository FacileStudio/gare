package domains

import (
	"net/http"
	"strings"
	"testing"

	"github.com/FacileStudio/gare/apps/api/internal/apikit/apitest"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
	"github.com/go-chi/chi/v5"
)

// TestDomainsAddListRemove walks the routes over two static apps: a hostname is added, another
// app cannot take it, an invalid one is refused, and removing it leaves [] behind.
func TestDomainsAddListRemove(t *testing.T) {
	f := apitest.New(t)
	f.AddApp(t, &storage.AppConfig{Name: "demo", AppType: "static", Branch: "main"})
	f.AddApp(t, &storage.AppConfig{Name: "other", AppType: "static", Branch: "main"})
	srv := apitest.Serve(t, func(r chi.Router) { RegisterRoutes(r, NewService(f.Deps), apitest.Pass) })

	if _, body := apitest.Do(t, srv, http.MethodGet, "/api/apps/demo/domains", ""); strings.TrimSpace(body) != `{"domains":[]}` {
		t.Fatalf("empty list = %s", body)
	}
	status, body := apitest.Do(t, srv, http.MethodPost, "/api/apps/demo/domains", `{"hostname":"Demo.Example.test"}`)
	if status != http.StatusOK || strings.TrimSpace(body) != `{"domains":["demo.example.test"]}` {
		t.Fatalf("add = %d %s", status, body)
	}
	if status, body := apitest.Do(t, srv, http.MethodPost, "/api/apps/other/domains", `{"hostname":"demo.example.test"}`); status != http.StatusConflict {
		t.Fatalf("taken hostname = %d %s, want 409", status, body)
	}
	if status, _ := apitest.Do(t, srv, http.MethodPost, "/api/apps/demo/domains", `{"hostname":"bad host"}`); status != http.StatusBadRequest {
		t.Fatalf("invalid hostname = %d, want 400", status)
	}
	if status, _ := apitest.Do(t, srv, http.MethodDelete, "/api/apps/demo/domains/nope.example.test", ""); status != http.StatusNotFound {
		t.Fatalf("removing a hostname the app lacks = %d, want 404", status)
	}
	status, body = apitest.Do(t, srv, http.MethodDelete, "/api/apps/demo/domains/demo.example.test", "")
	if status != http.StatusOK || strings.TrimSpace(body) != `{"domains":[]}` {
		t.Fatalf("remove = %d %s", status, body)
	}
}
