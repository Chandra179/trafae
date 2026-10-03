package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
)

// migrationsFS holds the SQL migration scripts, applied in filename order and
// tracked in a goose_db_version table.
//
//go:embed migrations/sqlite/*.sql
var migrationsFS embed.FS

// Migrate applies every pending database migration. The server runs it on
// boot so a fresh deployment is searchable without a separate migrate step;
// server/cmd/migrate exposes the same path for operators who apply schema
// changes explicitly. Applying to a database created by older versions,
// whose tables were created outside goose, is safe: the catalog migrations
// are written with CREATE TABLE IF NOT EXISTS.
func Migrate(ctx context.Context, db *sql.DB) error {
	goose.SetBaseFS(migrationsFS)
	defer goose.SetBaseFS(nil)
	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("select migration dialect: %w", err)
	}
	if err := goose.UpContext(ctx, db, "migrations/sqlite"); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}
