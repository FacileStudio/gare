package auth

import (
	"context"
	"errors"
	"time"

	"github.com/FacileStudio/porte"
	troncerrors "github.com/FacileStudio/tronc/errors"
)

// refusedMessage is what a refused sign-in carries to /login?error=. porte
// forwards a permission_denied message as it is, so it is written for people.
const refusedMessage = "this account is not allowed to use gare"

// errNoCLILogin is why gare refuses to mint a CLI login code: the daemon has
// no CLI sign-in, so it keeps no table for codes.
var errNoCLILogin = errors.New("gare has no CLI sign-in")

// gatedUsers is the porte.UserStore porte/oidc writes through on every
// sign-in, so the allowlist is enforced at the one point porte asks the app to
// accept an account. porte's upsert commits the account and the identity link
// in its own transaction before the gate can refuse, so forget takes back the
// rows a refused login created. A refusal still leaves what existed before
// it: an account or link that was already there keeps its row, and porte's
// profile refresh of it stays applied. It never gets a session.
type gatedUsers struct {
	inner      accountStore
	identities identityFinder
	forget     func(context.Context, loginRows) error
	allow      allowlist
}

// UpsertFromOIDC runs porte's own upsert first and decides afterwards, so a
// refusal walks the same lookups and writes as an acceptance before its
// cleanup. Only a verified address counts: an unverified claim is whatever the
// person typed.
func (g gatedUsers) UpsertFromOIDC(ctx context.Context, claims porte.Claims) (int64, error) {
	rows, err := g.before(ctx, claims)
	if err != nil {
		return 0, err
	}
	rows.userID, err = g.inner.UpsertFromOIDC(ctx, claims)
	if err != nil {
		return 0, err
	}
	if g.allow.allows(claims.Email) && claims.EmailVerified {
		return rows.userID, nil
	}
	if err := g.forget(ctx, rows); err != nil {
		return 0, err
	}
	return 0, troncerrors.Forbidden(refusedMessage)
}

// noLoginCodes is the porte.LoginCodeStore of a daemon without CLI sign-in.
// porte/oidc requires one once OIDC is on; this one refuses to mint a code and
// finds none, so the CLI flow fails closed instead of reaching a table the
// migrations deliberately do not create.
type noLoginCodes struct{}

// Create refuses every code, which ends a CLI login on porte's failure page.
func (noLoginCodes) Create(context.Context, porte.LoginCode) error {
	return errNoCLILogin
}

// Consume finds no code, which porte answers with its usual 401.
func (noLoginCodes) Consume(context.Context, string) (porte.LoginCode, error) {
	return porte.LoginCode{}, porte.ErrNotFound
}

// DeleteExpired has nothing to sweep.
func (noLoginCodes) DeleteExpired(context.Context, time.Time) (int64, error) {
	return 0, nil
}
