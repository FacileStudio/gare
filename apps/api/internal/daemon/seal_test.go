package daemon

import (
	"strings"
	"testing"
)

// TestTokensAreSealedAtRestWithAKey proves PORTE_TOKEN_KEY decides whether
// the provider's access token reaches porte_identities as issued or sealed.
func TestTokensAreSealedAtRestWithAKey(t *testing.T) {
	h := newHarness(t, "", "ops@example.test")
	h.signIn("ops@example.test")
	if stored := h.accessToken(); stored != "access" {
		t.Fatalf("access token without a key = %q, want the plaintext the provider issued", stored)
	}
	h.cfg.Auth.TokenKey = strings.Repeat("ab", 32)
	h.restart("ops@example.test")
	h.signIn("ops@example.test")
	if stored := h.accessToken(); !strings.HasPrefix(stored, "pv1:") || strings.Contains(stored, "access") {
		t.Fatalf("access token with a key = %q, want a sealed pv1: value", stored)
	}
}

// accessToken reads the one identity row's stored access token.
func (h *harness) accessToken() string {
	h.t.Helper()
	var stored string
	if err := h.db.QueryRow("SELECT access_token FROM porte_identities").Scan(&stored); err != nil {
		h.t.Fatalf("read access token: %v", err)
	}
	return stored
}
