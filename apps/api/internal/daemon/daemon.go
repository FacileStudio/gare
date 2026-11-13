package daemon

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/FacileStudio/gare/apps/api/internal/server"
	"github.com/FacileStudio/gare/apps/api/migrations"
	"github.com/FacileStudio/gare/apps/api/modules/auth"
	"github.com/FacileStudio/tronc/apiref"
	"github.com/FacileStudio/tronc/spa"
	"github.com/go-chi/chi/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// Daemon is the assembled API: the migrated database, the sign-in module and
// the router that serves them beside the webhook intake.
type Daemon struct {
	db      *sql.DB
	auth    *auth.Service
	handler http.Handler
}

// New opens and migrates the database, then builds the router. A migration
// that fails is returned, so the process exits non-zero before it serves a
// request against a schema that did not apply.
func New(ctx context.Context, cfg Config, intake *server.Server, logger *slog.Logger) (*Daemon, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open the database: %w", err)
	}
	handler, service, err := assemble(ctx, cfg, db, intake, logger)
	if err != nil {
		return nil, errors.Join(err, db.Close())
	}
	return &Daemon{db: db, auth: service, handler: handler}, nil
}

// assemble migrates db and wires the routes over it.
func assemble(ctx context.Context, cfg Config, db *sql.DB, intake *server.Server, logger *slog.Logger) (http.Handler, *auth.Service, error) {
	if err := migrations.Run(ctx, db, logger); err != nil {
		return nil, nil, fmt.Errorf("failed to migrate the database: %w", err)
	}
	authCfg := cfg.Auth
	authCfg.Logger = logger
	service, err := auth.New(ctx, db, authCfg)
	if err != nil {
		return nil, nil, err
	}
	router, err := intake.Router(server.Routes{
		API:       func(api chi.Router) error { return auth.RegisterRoutes(api, service) },
		Docs:      Reference(),
		ClientDir: cfg.ClientDir,
		Logger:    logger,
	})
	return router, service, err
}

// Reference is the API reference the daemon serves at /docs.
func Reference() apiref.Config {
	return apiref.Config{
		Title:    "Gare",
		Registry: apiref.Registry{Modules: []apiref.Module{auth.Documentation()}},
	}
}

// logClient says at start whether the web UI is served and from where, since
// a directory with no index.html turns the UI off without any other sign.
func logClient(logger *slog.Logger, dir string) {
	if spa.Available(dir) {
		logger.Info("web UI served", slog.String("client_dir", dir))
		return
	}
	logger.Warn("web UI off: no index.html in CLIENT_DIR", slog.String("client_dir", dir))
}

// Handler is the daemon's whole HTTP surface.
func (d *Daemon) Handler() http.Handler { return d.handler }

// Close releases the database.
func (d *Daemon) Close() error { return d.db.Close() }

// Serve listens on cfg.Listen until ctx ends, then stops taking requests and
// lets the webhook intake finish the deploys it is running.
func Serve(ctx context.Context, cfg Config, intake *server.Server, logger *slog.Logger) error {
	d, err := New(ctx, cfg, intake, logger)
	if err != nil {
		return err
	}
	defer d.Close()
	go d.auth.Run(ctx)

	httpServer := &http.Server{Addr: cfg.Listen, Handler: d.handler, ReadHeaderTimeout: 10 * time.Second}
	served := make(chan error, 1)
	go func() { served <- httpServer.ListenAndServe() }()
	logger.Info("gare daemon listening", slog.String("addr", cfg.Listen))
	logClient(logger, cfg.ClientDir)

	select {
	case err := <-served:
		return err
	case <-ctx.Done():
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return errors.Join(httpServer.Shutdown(shutdownCtx), intake.Shutdown(shutdownCtx))
}
