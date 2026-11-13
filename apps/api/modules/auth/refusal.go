package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/FacileStudio/porte"
)

// accountStore is porte's UserStore plus the address lookup the gate needs to
// tell a new account from one that was already there before a sign-in.
type accountStore interface {
	porte.UserStore
	FindByEmail(ctx context.Context, email string) (int64, error)
}

// identityFinder is the one IdentityStore method the gate reads, to learn
// whether a sign-in's identity link already existed.
type identityFinder interface {
	Find(ctx context.Context, provider, subject string) (porte.StoredIdentity, error)
}

// loginRows is what a sign-in found before porte's upsert ran: which of the
// identity link and the account were absent, and so are this login's own work.
type loginRows struct {
	userID     int64
	provider   string
	subject    string
	newLink    bool
	newAccount bool
}

// before records which rows a sign-in is about to create. porte commits them
// in its own transaction ahead of the gate's verdict, so this is the only
// moment the gate can learn what a refusal must take back.
func (g gatedUsers) before(ctx context.Context, claims porte.Claims) (loginRows, error) {
	rows := loginRows{provider: claims.Provider, subject: claims.Subject}
	_, err := g.identities.Find(ctx, claims.Provider, claims.Subject)
	if rows.newLink, err = absent(err); err != nil {
		return rows, err
	}
	_, err = g.inner.FindByEmail(ctx, normalizeEmail(claims.Email))
	rows.newAccount, err = absent(err)
	return rows, err
}

// absent turns a lookup's error into whether the row was missing, keeping
// every other failure an error so a broken database never reads as "new".
func absent(err error) (bool, error) {
	switch {
	case err == nil:
		return false, nil
	case errors.Is(err, porte.ErrNotFound):
		return true, nil
	}
	return false, err
}

// forgetRefused removes what a refused sign-in created, in one transaction:
// the identity link when this login made it, then the account when this login
// made it and it now holds no identity and no session. A row that existed
// before the login is never touched.
func forgetRefused(db *sql.DB) func(context.Context, loginRows) error {
	return func(ctx context.Context, rows loginRows) error {
		if !rows.newLink {
			return nil
		}
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("forget refused sign-in: %w", err)
		}
		defer func() { _ = tx.Rollback() }()
		if err := deleteRefusedRows(ctx, tx, rows); err != nil {
			return fmt.Errorf("forget refused sign-in: %w", err)
		}
		return tx.Commit()
	}
}

// deleteRefusedRows is forgetRefused's two statements. The account delete
// re-checks identities and sessions itself, so a concurrent sign-in that
// linked or used the account in the meantime keeps it.
func deleteRefusedRows(ctx context.Context, tx *sql.Tx, rows loginRows) error {
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM porte_identities WHERE provider = $1 AND subject = $2 AND user_id = $3`,
		rows.provider, rows.subject, rows.userID); err != nil {
		return err
	}
	if !rows.newAccount {
		return nil
	}
	_, err := tx.ExecContext(ctx, `
		DELETE FROM porte_users u
		 WHERE u.id = $1
		   AND NOT EXISTS (SELECT 1 FROM porte_identities i WHERE i.user_id = u.id)
		   AND NOT EXISTS (SELECT 1 FROM porte_sessions s WHERE s.user_id = u.id)`, rows.userID)
	return err
}
