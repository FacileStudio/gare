// Package migrations embeds the ordered SQL that owns the daemon's schema and
// applies it through tronc/migrate.
//
// It is a package so the tests run the same migrations the binary does.
package migrations

import (
	"context"
	"database/sql"
	"embed"
	"io/fs"
	"log/slog"

	"github.com/FacileStudio/tronc/migrate"
)

//go:embed *.sql
var files embed.FS

// FS returns the migration files at their root, the layout tronc/migrate
// expects. It is a function rather than an exported variable so no caller can
// swap the set the binary ships.
func FS() fs.FS {
	return files
}

// Config is the tronc/migrate configuration over the embedded files, shared by
// Run and by the `migrate` subcommand the server wires through migrate.Command.
func Config(db *sql.DB, logger *slog.Logger) migrate.Config {
	return migrate.Config{DB: db, FS: files, Logger: logger}
}

// Run applies every pending migration behind tronc/migrate's advisory lock.
// The daemon calls it at start and exits non-zero on its error, so a schema
// that did not apply never serves a request.
func Run(ctx context.Context, db *sql.DB, logger *slog.Logger) error {
	return migrate.Run(ctx, Config(db, logger))
}
