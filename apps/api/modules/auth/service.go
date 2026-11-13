package auth

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/FacileStudio/porte/directory"
	"github.com/FacileStudio/porte/kit"
	"github.com/FacileStudio/porte/oidc"
	"github.com/FacileStudio/porte/pg"
	"github.com/FacileStudio/porte/session"
	troncerrors "github.com/FacileStudio/tronc/errors"
)

// revokedMessage is the 401 a session gets once its address leaves the list.
const revokedMessage = "this account is no longer allowed to use gare"

// Service holds porte's session manager and OIDC kit, built the way Antenne
// builds them, plus the allowlist every sign-in and every request is held to.
type Service struct {
	sessions   *session.Manager
	oidc       *oidc.Kit
	accounts   *kit.Store
	deletions  *directory.Deletions
	allow      allowlist
	controller controller
}

// New builds the module over the migrated database. It performs OIDC
// discovery, so an unreachable or half-configured issuer fails the boot rather
// than the first sign-in.
func New(ctx context.Context, db *sql.DB, cfg Config) (*Service, error) {
	store := pg.New(db)
	identities := store.Identities()
	sessions, err := session.New(cfg.Porte, session.Deps{Sessions: store.Sessions(), Logger: cfg.Logger})
	if err != nil {
		return nil, err
	}
	service := &Service{
		sessions: sessions,
		accounts: &kit.Store{DB: db, Columns: kit.Columns{Table: "porte_users"}, Logger: cfg.Logger},
		allow:    newAllowlist(cfg.AdminEmails),
	}
	service.deletions = newDeletions(cfg, store, service.accounts)
	service.oidc, err = oidc.New(ctx, cfg.Porte, oidc.Deps{
		Users:      gatedUsers{inner: store.Users(), identities: identities, forget: forgetRefused(db), allow: service.allow},
		Identities: identities,
		Codes:      noLoginCodes{},
		Sessions:   sessions,
		Logger:     cfg.Logger,
		Erased:     service.deletions,
	})
	if err != nil {
		return nil, err
	}
	return service, nil
}

// newDeletions builds the receiver for the people the provider erased. The
// account is porte_users itself, so erasing it is kit's EraseAccount over that
// table; the kit store is used only to read and delete, never to write an
// account, which is why it is a literal and not kit.NewStore.
func newDeletions(cfg Config, store *pg.Store, accounts *kit.Store) *directory.Deletions {
	return &directory.Deletions{
		Client: directory.Client{
			Issuer:       cfg.Porte.Issuer,
			ClientID:     cfg.Porte.ClientID,
			ClientSecret: cfg.Porte.ClientSecret,
		},
		OnPersonDeleted: accounts.EraseAccount,
		Erase:           store.EraseIdentity,
		Holds:           store.HoldsIdentity,
		State:           fileState{path: cfg.StatePath},
		Logger:          cfg.Logger,
	}
}

// Run polls the provider for erased people until ctx ends. Run it once per
// daemon: the receiver's state is one file.
func (s *Service) Run(ctx context.Context) {
	s.deletions.Run(ctx)
}

// Identity resolves the session's user and holds it to the allowlist, so a
// session whose address was removed from GARE_ADMIN_EMAILS stops working on
// its next request instead of living out its thirty days.
func (s *Service) Identity(ctx context.Context, userID int64) (kit.Identity, error) {
	identity, err := s.accounts.Identity(ctx, userID)
	if err != nil {
		return kit.Identity{}, err
	}
	if !s.allow.allows(identity.Email) {
		return kit.Identity{}, troncerrors.Unauthorized(revokedMessage)
	}
	return identity, nil
}

// RequireAuth is the middleware every operator route sits behind: porte
// authenticates the cookie or bearer and enforces X-Facile-CSRF on cookie
// mutations, then Identity applies the allowlist.
func (s *Service) RequireAuth() func(http.Handler) http.Handler {
	return kit.RequireAuth(s.sessions, s)
}
