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
// accept an account. A refused address gets no identity row and no session,
// because porte stops at the error before writing either.
type gatedUsers struct {
	inner porte.UserStore
	allow allowlist
}

// UpsertFromOIDC runs porte's own upsert first and decides afterwards, so a
// refusal does the same database work as an acceptance and its timing does
// not tell an outsider which addresses are on the list. Only a verified
// address counts: an unverified claim is whatever the person typed.
func (g gatedUsers) UpsertFromOIDC(ctx context.Context, claims porte.Claims) (int64, error) {
	userID, err := g.inner.UpsertFromOIDC(ctx, claims)
	allowed := g.allow.allows(claims.Email) && claims.EmailVerified
	if err != nil {
		return 0, err
	}
	if !allowed {
		return 0, troncerrors.Forbidden(refusedMessage)
	}
	return userID, nil
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
