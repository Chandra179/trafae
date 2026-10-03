// Command migrate applies the database schema migrations for the configured
// environment and exits. The server runs the same migrations on boot; this
// command exists for operators who apply schema changes explicitly.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Chandra179/trafae/server/config"
	"github.com/Chandra179/trafae/server/store"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()
	cfg, err := config.Load(config.Path(config.Environment()))
	if err != nil {
		return err
	}
	db, err := store.NewSQLite(ctx, cfg.SQLite.DSN)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()
	return store.Migrate(ctx, db)
}
