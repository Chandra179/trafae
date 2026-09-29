package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

// NewSQLite opens an embedded SQLite database (a single file on disk) and
// verifies connectivity before returning. No server or container required.
func NewSQLite(ctx context.Context, dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", sqliteDSN(dsn))
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	return db, nil
}

// sqliteDSN appends the connection pragmas the server needs. busy_timeout
// makes writers wait instead of failing with SQLITE_BUSY while the Project
// Gutenberg catalog refresh holds the write lock, and WAL lets readers keep
// serving searches during it. Both pragmas apply per connection, so they must
// ride on the DSN rather than be executed once on a pooled connection.
func sqliteDSN(dsn string) string {
	dsn = strings.TrimSpace(dsn)
	separator := "?"
	if strings.Contains(dsn, "?") {
		separator = "&"
	}
	return dsn + separator + "_busy_timeout=10000&_journal_mode=WAL"
}
