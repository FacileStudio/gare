package daemon

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/FacileStudio/gare/apps/api/internal/server"
	"github.com/FacileStudio/gare/apps/api/modules/auth"
	"github.com/FacileStudio/porte"
)

// databaseURLVar names the Postgres the daemon tests run against.
const databaseURLVar = "GARE_TEST_DATABASE_URL"

// hookSecret is the webhook secret every harness intake verifies against.
const hookSecret = "hook-secret"

// harness is one daemon over a scratch schema, reached through a real
// listener and a browser-like client that keeps cookies and stops at every
// redirect so a test can read each Location.
type harness struct {
	t        *testing.T
	idp      *fakeIdP
	base     string
	cfg      Config
	db       *sql.DB
	handler  atomic.Pointer[http.Handler]
	client   *http.Client
	deployed chan string
}

// newHarness starts the provider, the listener and a daemon allowing emails.
func newHarness(t *testing.T, clientDir string, emails ...string) *harness {
	t.Helper()
	dsn := scratchURL(t)
	h := &harness{t: t, idp: newFakeIdP(t), deployed: make(chan string, 4)}
	listener := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		(*h.handler.Load()).ServeHTTP(w, r)
	}))
	t.Cleanup(listener.Close)
	h.base = listener.URL
	h.cfg = Config{DatabaseURL: dsn, ClientDir: clientDir, Auth: auth.Config{
		Porte: porte.Config{
			Issuer: h.idp.server.URL, ClientID: "gare", ClientSecret: "client-secret",
			RedirectURL: h.base + "/api/auth/oidc/callback", SuccessURL: h.base + "/", SSOOnly: true,
		},
		StatePath: t.TempDir() + "/deletions.json",
	}}
	jar, _ := cookiejar.New(nil)
	h.client = &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	h.restart(emails...)
	h.db = openDB(t, dsn)
	return h
}

// restart replaces the daemon on the same listener and database with one
// allowing emails, the way an operator edits GARE_ADMIN_EMAILS and restarts.
func (h *harness) restart(emails ...string) {
	h.t.Helper()
	cfg := h.cfg
	cfg.Auth.AdminEmails = emails
	intake := server.New(server.Config{Secret: hookSecret, DeployHandler: func(_ context.Context, app string) error {
		h.deployed <- app
		return nil
	}})
	d, err := New(context.Background(), cfg, intake, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		h.t.Fatalf("daemon: %v", err)
	}
	h.t.Cleanup(func() {
		if err := d.Close(); err != nil {
			h.t.Errorf("close daemon: %v", err)
		}
	})
	handler := d.Handler()
	h.handler.Store(&handler)
}

// signIn walks the browser through the provider and returns where the
// callback finally sent it.
func (h *harness) signIn(email string) string {
	h.t.Helper()
	h.idp.setEmail(email)
	next := h.base + "/api/auth/oidc"
	for range 3 {
		response := h.do(http.MethodGet, next, nil)
		if response.StatusCode != http.StatusFound {
			h.t.Fatalf("GET %s = %d, want 302", next, response.StatusCode)
		}
		next = response.Header.Get("Location")
	}
	return next
}

// do sends one request with the harness's cookies and the given headers.
func (h *harness) do(method, target string, header http.Header) *http.Response {
	h.t.Helper()
	request, err := http.NewRequest(method, target, strings.NewReader(""))
	if err != nil {
		h.t.Fatalf("request: %v", err)
	}
	for key, values := range header {
		request.Header[key] = values
	}
	response, err := h.client.Do(request)
	if err != nil {
		h.t.Fatalf("%s %s: %v", method, target, err)
	}
	h.t.Cleanup(func() {
		if err := response.Body.Close(); err != nil {
			h.t.Errorf("close body: %v", err)
		}
	})
	return response
}

// sessions counts porte's session rows in the scratch schema.
func (h *harness) sessions() int {
	return h.count("porte_sessions")
}

// count is the number of rows in one of porte's tables in the scratch schema.
func (h *harness) count(table string) int {
	h.t.Helper()
	var n int
	if err := h.db.QueryRow("SELECT count(*) FROM " + table).Scan(&n); err != nil {
		h.t.Fatalf("count %s: %v", table, err)
	}
	return n
}

// scratchURL creates a schema of its own for the test, dropped afterwards,
// and returns a URL whose connections land in it. Without a database the
// test skips loudly.
func scratchURL(t *testing.T) string {
	t.Helper()
	raw := os.Getenv(databaseURLVar)
	if raw == "" {
		fmt.Fprintf(os.Stderr, "daemon: %s is unset, skipping the Postgres test\n", databaseURLVar)
		t.Skipf("%s is unset", databaseURLVar)
	}
	schema := fmt.Sprintf("gare_daemon_%d", time.Now().UnixNano())
	exec(t, raw, "CREATE SCHEMA "+schema)
	t.Cleanup(func() { exec(t, raw, "DROP SCHEMA "+schema+" CASCADE") })
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse %s: %v", databaseURLVar, err)
	}
	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

// openDB connects and closes the pool when the test ends.
func openDB(t *testing.T, dsn string) *sql.DB {
	t.Helper()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close: %v", err)
		}
	})
	return db
}

// exec runs one statement on a connection of its own, so creating and
// dropping the scratch schema never shares a pool with the daemon under test.
func exec(t *testing.T, dsn, statement string) {
	t.Helper()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(statement); err != nil {
		t.Fatalf("%s: %v", statement, err)
	}
}
