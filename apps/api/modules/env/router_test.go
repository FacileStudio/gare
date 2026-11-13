package env

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/FacileStudio/gare/apps/api/internal/apikit/apitest"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
	"github.com/go-chi/chi/v5"
)

// serve mounts the module over a fixture holding one static app named demo.
func serve(t *testing.T) *httptest.Server {
	t.Helper()
	f := apitest.New(t)
	f.AddApp(t, &storage.AppConfig{Name: "demo", AppType: "static", Branch: "main"})
	return apitest.Serve(t, func(r chi.Router) { RegisterRoutes(r, NewService(f.Deps), apitest.Pass) })
}

// TestMask proves only a long value shows its ends, and no mask tells a length.
func TestMask(t *testing.T) {
	cases := map[string]string{
		"":                  "••••••••",
		"short":             "••••••••",
		"elevenchars":       "••••••••",
		"twelve-chars":      "tw••••••••rs",
		"a much longer one": "a ••••••••ne",
	}
	for value, want := range cases {
		if got := mask(value); got != want {
			t.Errorf("mask(%q) = %q, want %q", value, got, want)
		}
	}
}

// TestPatchMergesAndGetMasks walks the routes: a set lands masked and sorted, a reveal answers one
// value, an unset removes only its key, and a bad key or unknown variable is refused.
func TestPatchMergesAndGetMasks(t *testing.T) {
	srv := serve(t)
	status, body := apitest.Do(t, srv, http.MethodPatch, "/api/apps/demo/env",
		`{"set":{"SHORT":"abc","LONG":"abcdefghijklmnop"}}`)
	want := `{"entries":[{"key":"LONG","masked":"ab••••••••op"},{"key":"SHORT","masked":"••••••••"}]}`
	if status != http.StatusOK || strings.TrimSpace(body) != want {
		t.Fatalf("PATCH set = %d %s", status, body)
	}
	if _, body := apitest.Do(t, srv, http.MethodGet, "/api/apps/demo/env?reveal=LONG", ""); !strings.Contains(body, `"value":"abcdefghijklmnop"`) {
		t.Fatalf("reveal = %s", body)
	}
	_, body = apitest.Do(t, srv, http.MethodPatch, "/api/apps/demo/env", `{"unset":["SHORT"]}`)
	if strings.TrimSpace(body) != `{"entries":[{"key":"LONG","masked":"ab••••••••op"}]}` {
		t.Fatalf("PATCH unset = %s", body)
	}
	refused := map[string]string{
		`{"set":{"1BAD":"x"}}`:            "PATCH",
		`{"set":{"A":"x"},"unset":["A"]}`: "PATCH",
		"/api/apps/demo/env?reveal=SHORT": http.MethodGet,
		"/api/apps/nope/env":              http.MethodGet,
	}
	for input, method := range refused {
		path, body := input, ""
		if method == "PATCH" {
			path, body = "/api/apps/demo/env", input
		}
		if status, answer := apitest.Do(t, srv, method, path, body); status < 400 {
			t.Errorf("%s %s %s = %d %s, want a refusal", method, path, body, status, answer)
		}
	}
}
