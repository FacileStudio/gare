package daemon

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FacileStudio/gare/apps/api/internal/deploys"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
	"github.com/FacileStudio/tronc/apiref"
	"github.com/go-chi/chi/v5"
)

// csrf is the header the client sends on every mutation.
var csrf = http.Header{"X-Facile-Csrf": {"1"}}

// TestMeNeedsASession proves the API refuses a caller with no session.
func TestMeNeedsASession(t *testing.T) {
	h := newHarness(t, "", "ops@example.test")
	if response := h.do(http.MethodGet, h.base+"/api/me", nil); response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("GET /api/me = %d, want 401", response.StatusCode)
	}
}

// TestAnAddressOffTheListIsRefusedWithoutASession proves the allowlist sends a
// stranger back to the login page and leaves no account, identity link or
// session for them, while an operator who was there first keeps their rows.
func TestAnAddressOffTheListIsRefusedWithoutASession(t *testing.T) {
	h := newHarness(t, "", "ops@example.test")
	h.signIn("ops@example.test")
	landed := h.signIn("stranger@example.test")
	parsed, err := url.Parse(landed)
	if err != nil || parsed.Path != "/login" || parsed.Query().Get("error") == "" {
		t.Fatalf("refused login landed on %q, want /login?error=...", landed)
	}
	assertRows(t, h, 1, 1, 1)
	h.restart("someone-else@example.test")
	h.signIn("ops@example.test")
	assertRows(t, h, 1, 1, 1)
}

// assertRows checks porte's account, identity and session row counts.
func assertRows(t *testing.T, h *harness, users, identities, sessions int) {
	t.Helper()
	got := [3]int{h.count("porte_users"), h.count("porte_identities"), h.sessions()}
	if want := [3]int{users, identities, sessions}; got != want {
		t.Fatalf("users, identities, sessions = %v, want %v", got, want)
	}
}

// TestAnAllowedOperatorSignsInAndOut walks the whole session: sign in with an
// address in another case, read /api/me bare, then the CSRF rule and an
// idempotent logout.
func TestAnAllowedOperatorSignsInAndOut(t *testing.T) {
	h := newHarness(t, "", " Ops@Example.test ")
	if landed := h.signIn("ops@example.test"); landed != h.base+"/" {
		t.Fatalf("allowed login landed on %q, want %q", landed, h.base+"/")
	}
	if n := h.sessions(); n != 1 {
		t.Fatalf("session rows = %d, want 1", n)
	}
	response := h.do(http.MethodGet, h.base+"/api/me", nil)
	body, _ := io.ReadAll(response.Body)
	var me map[string]any
	if response.StatusCode != http.StatusOK || json.Unmarshal(body, &me) != nil {
		t.Fatalf("GET /api/me = %d %s", response.StatusCode, body)
	}
	if me["email"] != "ops@example.test" || me["name"] != "Ops Person" || me["picture"] != nil || len(me) != 3 {
		t.Fatalf("GET /api/me = %s, want bare {email, name, picture: null}", body)
	}
	assertLogout(t, h)
}

// assertLogout proves a cookie mutation needs X-Facile-CSRF, that logout
// with it ends the session, and that a second logout is just as fine.
func assertLogout(t *testing.T, h *harness) {
	t.Helper()
	if response := h.do(http.MethodPost, h.base+"/api/auth/logout", nil); response.StatusCode != http.StatusForbidden {
		t.Fatalf("cookie POST without X-Facile-CSRF = %d, want 403", response.StatusCode)
	}
	for attempt := 1; attempt <= 2; attempt++ {
		if response := h.do(http.MethodPost, h.base+"/api/auth/logout", csrf); response.StatusCode != http.StatusOK {
			t.Fatalf("logout %d = %d, want 200", attempt, response.StatusCode)
		}
	}
	if n := h.sessions(); n != 0 {
		t.Fatalf("session rows after logout = %d, want 0", n)
	}
}

// TestLeavingTheListEndsTheSession proves a session stops working once its
// address is removed from GARE_ADMIN_EMAILS and the daemon restarts, and that
// the refusal revokes it: putting the address back does not revive the cookie.
func TestLeavingTheListEndsTheSession(t *testing.T) {
	h := newHarness(t, "", "ops@example.test")
	h.signIn("ops@example.test")
	if response := h.do(http.MethodGet, h.base+"/api/me", nil); response.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/me before = %d, want 200", response.StatusCode)
	}
	h.restart("someone-else@example.test")
	if response := h.do(http.MethodGet, h.base+"/api/me", nil); response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("GET /api/me after removal = %d, want 401", response.StatusCode)
	}
	if n := h.sessions(); n != 0 {
		t.Fatalf("session rows after removal = %d, want 0", n)
	}
	h.restart("ops@example.test")
	if response := h.do(http.MethodGet, h.base+"/api/me", nil); response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("GET /api/me with the old cookie after re-adding = %d, want 401", response.StatusCode)
	}
}

// TestWebhookIntakeIsUnchanged proves the root routes still reach the webhook
// handler: health, a refused signature, and an accepted trigger, which the
// daemon's engine records as a webhook deploy of the app.
func TestWebhookIntakeIsUnchanged(t *testing.T) {
	h := newHarness(t, "", "ops@example.test")
	if err := h.cfg.Store.Save(&storage.AppConfig{Name: "demo", AppType: "static", Branch: "main"}); err != nil {
		t.Fatal(err)
	}
	response := h.do(http.MethodGet, h.base+"/health", nil)
	if body, _ := io.ReadAll(response.Body); response.StatusCode != http.StatusOK || string(body) != "OK" {
		t.Fatalf("GET /health = %d %q", response.StatusCode, body)
	}
	bad := http.Header{"X-Gitlab-Token": {"wrong"}}
	if response := h.do(http.MethodPost, h.base+"/webhook/demo", bad); response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unsigned webhook = %d, want 401", response.StatusCode)
	}
	good := http.Header{"X-Gitlab-Token": {hookSecret}}
	if response := h.do(http.MethodPost, h.base+"/webhook/demo", good); response.StatusCode != http.StatusAccepted {
		t.Fatalf("signed webhook = %d, want 202", response.StatusCode)
	}
	list, err := deploys.Open(h.cfg.Store.AppDir("demo"), "demo").List()
	if err != nil || len(list) != 1 || list[0].Trigger != deploys.TriggerWebhook {
		t.Fatalf("history = %+v, %v; want one webhook deploy", list, err)
	}
}

// TestTheClientIsServedAtTheRootAndOnDeepLinks proves the SPA fallback, and
// that an unknown API path stays a JSON 404 instead of the client's page.
func TestTheClientIsServedAtTheRootAndOnDeepLinks(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<!doctype html><title>gare</title>"), 0o600); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, dir, "ops@example.test")
	for _, path := range []string{"/", "/apps/demo"} {
		response := h.do(http.MethodGet, h.base+path, nil)
		if body, _ := io.ReadAll(response.Body); response.StatusCode != http.StatusOK || !strings.Contains(string(body), "gare") {
			t.Fatalf("GET %s = %d %q, want the client", path, response.StatusCode, body)
		}
	}
	if response := h.do(http.MethodGet, h.base+"/api/nope", nil); response.StatusCode != http.StatusNotFound {
		t.Fatalf("GET /api/nope = %d, want 404", response.StatusCode)
	}
}

// TestEveryAPIRouteIsDocumented keeps /docs in step with the router.
func TestEveryAPIRouteIsDocumented(t *testing.T) {
	h := newHarness(t, "", "ops@example.test")
	router, ok := (*h.handler.Load()).(chi.Routes)
	if !ok {
		t.Fatal("the daemon handler is not a chi router")
	}
	if missing := apiref.Undocumented(router, Reference(), "/webhook"); len(missing) > 0 {
		t.Fatalf("undocumented routes: %v", missing)
	}
}
