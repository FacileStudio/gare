// Package auth signs operators in to the daemon through porte against the
// suite's identity provider, and lets in only the addresses GARE_ADMIN_EMAILS
// names. It owns no credential logic of its own: sessions, cookies, CSRF and
// the OIDC flow are porte's, and this package only decides who may hold one.
package auth

import (
	"log/slog"

	"github.com/FacileStudio/porte"
)

// Me is the signed-in operator as GET /api/me answers it. It is the bare shape
// the client reads, not porte's {user} envelope, and Picture is null rather
// than an empty string when the provider sent none.
type Me struct {
	Email   string  `json:"email"`
	Name    string  `json:"name"`
	Picture *string `json:"picture"`
}

// Config is what the module needs from the daemon's environment. AdminEmails
// is the allowlist, already split; an empty list lets nobody in. StatePath is
// the file that remembers which identities the provider erased, kept on disk
// beside the rest of gare's state so Postgres holds porte's tables only.
type Config struct {
	Porte       porte.Config
	AdminEmails []string
	StatePath   string
	Logger      *slog.Logger
}
