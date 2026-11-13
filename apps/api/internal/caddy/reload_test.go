package caddy

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A snippet gare wrote but could not hand to a running Caddy leaves that server on its previous
// configuration, so an undeliverable reload has to surface instead of passing as a success.
func TestReloadViaAdminAPIReportsAnUnreachableServer(t *testing.T) {
	if err := reloadViaAdminAPI(context.Background(), "http://127.0.0.1:1/load", writeTempCaddyfile(t)); err == nil {
		t.Error("expected an unreachable admin api to be reported")
	}
}

func TestReloadViaAdminAPIReportsAnErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad config", http.StatusBadRequest)
	}))
	defer srv.Close()

	err := reloadViaAdminAPI(context.Background(), srv.URL, writeTempCaddyfile(t))
	if err == nil || !strings.Contains(err.Error(), "400") {
		t.Errorf("expected the admin api status in the error, got %v", err)
	}
}

func TestReloadViaAdminAPIPostsTheCaddyfile(t *testing.T) {
	var method, contentType, body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		contentType = r.Header.Get("Content-Type")
		raw, _ := io.ReadAll(r.Body)
		body = string(raw)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	if err := reloadViaAdminAPI(context.Background(), srv.URL, writeTempCaddyfile(t)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if method != http.MethodPost {
		t.Errorf("got method %s, want POST", method)
	}
	if contentType != "text/caddyfile" {
		t.Errorf("got content type %q, want text/caddyfile", contentType)
	}
	if want := "example.com {\n\treverse_proxy localhost:8080\n}\n"; body != want {
		t.Errorf("got posted body %q, want %q", body, want)
	}
}

func writeTempCaddyfile(t *testing.T) string {
	t.Helper()
	content, err := GenerateSnippet([]string{"example.com"}, 8080)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "Caddyfile")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}
