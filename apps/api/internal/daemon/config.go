// Package daemon runs the API and the web client next to the webhook intake:
// it opens the database, migrates it, signs operators in through porte and
// serves everything on one listener. Only `gare server` uses it, so no other
// command ever needs a database.
package daemon

import (
	"errors"
	"net"
	"path/filepath"

	"github.com/FacileStudio/gare/apps/api/internal/engine"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
	"github.com/FacileStudio/gare/apps/api/internal/xdg"
	"github.com/FacileStudio/gare/apps/api/modules/auth"
	"github.com/FacileStudio/porte"
	troncenv "github.com/FacileStudio/tronc/env"
)

// DefaultListen is where the daemon listens when GARE_LISTEN is unset:
// loopback only, so a fresh install exposes nothing until the operator puts a
// proxy in front of it.
const DefaultListen = "127.0.0.1:9120"

// ListenEnv names the variable that sets the daemon's whole address, host
// included. It is the only way to bind anything other than loopback.
const ListenEnv = "GARE_LISTEN"

// Config is the daemon's environment. The OIDC names are the ones every suite
// app reads, so one deployment panel spells them the same way everywhere.
// ClientDir defaults under XDG_DATA_HOME rather than to a relative path,
// because a systemd user unit starts in $HOME and a relative default would
// silently find no client there. Store is where the apps live, the user's
// default store when nil, and Settings is what the engine runs every API and
// webhook operation with. allowLocalRepos lets a create clone a local path,
// which only tests turn on.
type Config struct {
	Listen          string
	DatabaseURL     string
	ClientDir       string
	Auth            auth.Config
	Store           *storage.Store
	Settings        engine.Settings
	allowLocalRepos bool
}

// Load reads the environment. It fails only on a value that is set but
// unusable; whether the API runs at all is Enabled's question.
func Load() (Config, error) {
	ssoOnly, err := troncenv.Bool("SSO_ONLY", true)
	if err != nil {
		return Config{}, err
	}
	return Config{
		Listen:      ListenFromEnv(""),
		DatabaseURL: troncenv.String("DATABASE_URL", ""),
		ClientDir:   troncenv.String("CLIENT_DIR", filepath.Join(xdg.DataHome(), "gare", "client")),
		Auth: auth.Config{
			Porte:       loadPorte(ssoOnly),
			AdminEmails: auth.ParseEmails(troncenv.String("GARE_ADMIN_EMAILS", "")),
			StatePath:   filepath.Join(xdg.DataHome(), "gare", "registre-deletions.json"),
			TokenKey:    troncenv.String(auth.TokenKeyEnv, ""),
		},
	}, nil
}

// ListenAddress resolves the address the daemon binds. An explicit listen
// address always wins, since it is the one setting that names a host;
// otherwise port, when given, replaces only the port of DefaultListen, so a
// --port flag never widens the bind to every interface.
func ListenAddress(listen, port string) string {
	if listen != "" {
		return listen
	}
	if port == "" {
		return DefaultListen
	}
	host, _, _ := net.SplitHostPort(DefaultListen)
	return net.JoinHostPort(host, port)
}

// ListenFromEnv is ListenAddress over GARE_LISTEN, shared by Load and by both
// modes of `gare server`, so every path resolves its address one way.
func ListenFromEnv(port string) string {
	return ListenAddress(troncenv.String(ListenEnv, ""), port)
}

// loadPorte reads porte's half of the environment under the suite's names.
func loadPorte(ssoOnly bool) porte.Config {
	return porte.Config{
		Issuer:       troncenv.String("OIDC_ISSUER", ""),
		ClientID:     troncenv.String("OIDC_CLIENT_ID", ""),
		ClientSecret: troncenv.String("OIDC_CLIENT_SECRET", ""),
		RedirectURL:  troncenv.String("OIDC_REDIRECT_URL", ""),
		SuccessURL:   troncenv.String("OIDC_SUCCESS_URL", ""),
		FailureURL:   troncenv.String("OIDC_FAILURE_URL", ""),
		SSOOnly:      ssoOnly,
	}
}

// Enabled reports whether the API and the web client run. They need a way to
// sign in, and gare signs in through OIDC only, so without OIDC_ISSUER the
// daemon stays the webhook intake it always was.
func (c Config) Enabled() bool {
	return c.Auth.Porte.Enabled()
}

// Validate refuses an enabled API that could not work: porte's own checks,
// a database to keep sessions in, and at least one address allowed in, since
// an empty list would refuse every sign-in without saying why.
func (c Config) Validate() error {
	if err := c.Auth.Porte.Validate(); err != nil {
		return err
	}
	if c.DatabaseURL == "" {
		return errors.New("DATABASE_URL is required when OIDC_ISSUER is set")
	}
	if len(c.Auth.AdminEmails) == 0 {
		return errors.New("GARE_ADMIN_EMAILS is required when OIDC_ISSUER is set: with no address listed, nobody can sign in")
	}
	return nil
}
