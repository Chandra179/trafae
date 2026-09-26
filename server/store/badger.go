package store

import (
	"fmt"

	badger "github.com/dgraph-io/badger/v4"
)

// NewBadger opens an embedded Badger key-value store in the given directory.
// Embedded like SQLite, so no server or container required.
func NewBadger(dir string) (*badger.DB, error) {
	opts := badger.DefaultOptions(dir)
	opts.Logger = nil

	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("open badger: %w", err)
	}

	return db, nil
}
