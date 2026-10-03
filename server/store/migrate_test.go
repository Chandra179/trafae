package store

import (
	"context"
	"path/filepath"
	"testing"
)

func TestMigrateCreatesSearchableSchema(t *testing.T) {
	t.Parallel()

	db, err := NewSQLite(context.Background(), filepath.Join(t.TempDir(), "migrate-test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	if err := Migrate(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"examples", "project_gutenberg_catalog", "project_gutenberg_catalog_sync", "goose_db_version"} {
		var name string
		if err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&name); err != nil {
			t.Fatalf("table %s missing after Migrate: %v", table, err)
		}
	}

	// Re-applying must be a no-op, which is also what makes migrating a
	// pre-goose database (tables already present) safe.
	if err := Migrate(context.Background(), db); err != nil {
		t.Fatalf("second Migrate call: %v", err)
	}
}
