package daemon

import (
	"context"
	"io"
	"log/slog"
	"reflect"
	"strings"
	"testing"

	"github.com/FacileStudio/gare/apps/api/internal/server"
)

// TestLoadWithoutOIDCKeepsTheAPIOff proves an install that sets nothing new
// stays a webhook intake, listening on loopback if it is ever enabled.
func TestLoadWithoutOIDCKeepsTheAPIOff(t *testing.T) {
	for _, name := range []string{"OIDC_ISSUER", "GARE_LISTEN", "DATABASE_URL", "GARE_ADMIN_EMAILS", "SSO_ONLY"} {
		t.Setenv(name, "")
	}
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Enabled() || cfg.Listen != DefaultListen || len(cfg.Auth.AdminEmails) != 0 || !cfg.Auth.Porte.SSOOnly {
		t.Fatalf("Load = %+v, want the API off on %s with SSO only and nobody allowed", cfg, DefaultListen)
	}
}

// TestNewFailsWithoutAMigratedDatabase proves the daemon refuses to build,
// and so `gare server` exits non-zero, when the schema cannot be applied.
func TestNewFailsWithoutAMigratedDatabase(t *testing.T) {
	cfg := Config{DatabaseURL: "postgres://gare:x@127.0.0.1:1/gare?sslmode=disable&connect_timeout=2"}
	cfg.Auth.Porte.Issuer = "https://sso.example.test"
	cfg.Auth.Porte.ClientID, cfg.Auth.Porte.ClientSecret = "gare", "secret"
	cfg.Auth.Porte.RedirectURL, cfg.Auth.Porte.SuccessURL = "https://g.example.test/cb", "https://g.example.test/"
	cfg.Auth.AdminEmails = []string{"ops@example.test"}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	if _, err := New(context.Background(), cfg, server.New(server.Config{}), logger); err == nil {
		t.Fatal("New succeeded against an unreachable database")
	}
}

// TestLoadReadsTheSuiteNames proves the OIDC block and gare's own variables
// land where porte and the daemon read them, and that an enabled API without
// a database is refused.
func TestLoadReadsTheSuiteNames(t *testing.T) {
	t.Setenv("OIDC_ISSUER", "https://sso.example.test")
	t.Setenv("OIDC_CLIENT_ID", "gare")
	t.Setenv("OIDC_CLIENT_SECRET", "secret")
	t.Setenv("OIDC_REDIRECT_URL", "https://gare.example.test/api/auth/oidc/callback")
	t.Setenv("OIDC_SUCCESS_URL", "https://gare.example.test/")
	t.Setenv("GARE_LISTEN", "10.0.0.1:9120")
	t.Setenv("GARE_ADMIN_EMAILS", "A@example.test, b@example.test")
	t.Setenv("DATABASE_URL", "")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.Enabled() || cfg.Listen != "10.0.0.1:9120" || cfg.Auth.Porte.ClientID != "gare" {
		t.Fatalf("Load = %+v", cfg)
	}
	if want := []string{"a@example.test", "b@example.test"}; !reflect.DeepEqual(cfg.Auth.AdminEmails, want) {
		t.Fatalf("AdminEmails = %q, want %q", cfg.Auth.AdminEmails, want)
	}
	if cfg.Validate() == nil {
		t.Fatal("an enabled API without DATABASE_URL passed validation")
	}
	cfg.DatabaseURL = "postgres://localhost/gare"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

// TestAnEnabledAPIWithNoAdminIsRefused proves the daemon will not start an
// API nobody can sign in to, and says which variable to set.
func TestAnEnabledAPIWithNoAdminIsRefused(t *testing.T) {
	cfg := Config{DatabaseURL: "postgres://localhost/gare"}
	cfg.Auth.Porte.Issuer = "https://sso.example.test"
	cfg.Auth.Porte.ClientID, cfg.Auth.Porte.ClientSecret = "gare", "secret"
	cfg.Auth.Porte.RedirectURL, cfg.Auth.Porte.SuccessURL = "https://g.example.test/cb", "https://g.example.test/"
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "GARE_ADMIN_EMAILS") {
		t.Fatalf("Validate with no admin address = %v, want an error naming GARE_ADMIN_EMAILS", err)
	}
	cfg.Auth.AdminEmails = []string{"ops@example.test"}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate with an admin address: %v", err)
	}
}
