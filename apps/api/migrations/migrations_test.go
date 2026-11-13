package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/FacileStudio/porte"
	portepg "github.com/FacileStudio/porte/pg"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// databaseURLVar names the Postgres the migration test runs against. Without
// it the test skips loudly, because a gate nobody notices is off is worse than
// none.
const databaseURLVar = "GARE_TEST_DATABASE_URL"

// TestRunCreatesPorteSchema proves the migrations apply to an empty database,
// apply again as a no-op, and leave a schema porte/pg can sign a user in on.
func TestRunCreatesPorteSchema(t *testing.T) {
	db := openScratch(t)
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	for attempt := 1; attempt <= 2; attempt++ {
		if err := Run(ctx, db, logger); err != nil {
			t.Fatalf("run %d: %v", attempt, err)
		}
	}
	store := portepg.New(db)
	userID, err := store.Users().UpsertFromOIDC(ctx, porte.Claims{
		Provider: "registre", Subject: "sub-1", Email: "Ops@Example.com", EmailVerified: true, Name: "Ops",
	})
	if err != nil {
		t.Fatalf("upsert user: %v", err)
	}
	created, err := store.Sessions().Create(ctx, porte.Session{TokenHash: "hash-1", UserID: userID})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	found, err := store.Sessions().Find(ctx, "hash-1")
	if err != nil || found.ID != created.ID || found.UserID != userID {
		t.Fatalf("find session = %+v, %v; want id %d for user %d", found, err, created.ID, userID)
	}
}

// openScratch opens the test database inside a schema of its own, so a re-run
// against the same database starts empty and leaves the public schema alone.
func openScratch(t *testing.T) *sql.DB {
	t.Helper()
	raw := os.Getenv(databaseURLVar)
	if raw == "" {
		fmt.Fprintf(os.Stderr, "migrations: %s is unset, skipping the Postgres test\n", databaseURLVar)
		t.Skipf("%s is unset", databaseURLVar)
	}
	schema := fmt.Sprintf("gare_test_%d", time.Now().UnixNano())
	exec(t, raw, "CREATE SCHEMA "+schema)
	t.Cleanup(func() { exec(t, raw, "DROP SCHEMA "+schema+" CASCADE") })
	return open(t, withSearchPath(t, raw, schema))
}

// exec runs one statement on a connection of its own, so creating and dropping
// the scratch schema never shares a pool with the migrations under test.
func exec(t *testing.T, dsn, statement string) {
	t.Helper()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(statement); err != nil {
		t.Fatalf("%s: %v", statement, err)
	}
}

// open connects through pgx and closes the pool when the test ends.
func open(t *testing.T, dsn string) *sql.DB {
	t.Helper()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close: %v", err)
		}
	})
	return db
}

// withSearchPath points every pooled connection at schema, where goose's
// unqualified ledger and porte's tables then land.
func withSearchPath(t *testing.T, raw, schema string) string {
	t.Helper()
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse %s: %v", databaseURLVar, err)
	}
	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}
