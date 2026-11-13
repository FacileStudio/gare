package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/FacileStudio/porte/pg"
)

// TokenKeyEnv names the variable holding the key porte seals the provider's
// access and refresh tokens with, the name every suite app reads.
const TokenKeyEnv = "PORTE_TOKEN_KEY"

// newStore builds porte's Postgres store, sealing the tokens porte_identities
// keeps when a key is configured. A key that is set but unreadable stops the
// boot; no key at all is allowed, and said out loud, because the tokens then
// sit in the database as the provider issued them.
func newStore(db *sql.DB, cfg Config) (*pg.Store, bool, error) {
	sealer, err := pg.ParseTokenSealer(cfg.TokenKey)
	if err != nil {
		return nil, false, fmt.Errorf("%s is set but invalid: %w", TokenKeyEnv, err)
	}
	if sealer == nil {
		cfg.Logger.Warn(TokenKeyEnv + " is unset: the identity provider's tokens are stored in plaintext")
	}
	return pg.New(db, pg.WithTokenSealer(sealer)), sealer != nil, nil
}

// reseal seals the tokens of rows written before the key was set, or under an
// older key, so turning sealing on covers people who do not sign in again.
func reseal(ctx context.Context, identities *pg.IdentityStore, logger *slog.Logger) {
	count, err := identities.ResealIdentities(ctx)
	if err != nil && !errors.Is(err, context.Canceled) {
		logger.Warn("porte token reseal failed", slog.Int("resealed", count), slog.Any("error", err))
		return
	}
	logger.Info("porte token reseal complete", slog.Int("resealed", count))
}
