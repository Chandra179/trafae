package example

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (d *dependencies) CreateExample(ctx context.Context, name string) (*Example, error) {
	if name == reservedName {
		return nil, fmt.Errorf("create example %q: %w", name, ErrReservedName)
	}

	ex, err := d.store.CreateExample(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("create example %q: %w", name, err)
	}

	return ex, nil
}

func (s *storeDependencies) CreateExample(ctx context.Context, name string) (*Example, error) {
	ex := &Example{ID: uuid.NewString(), Name: name}

	err := s.db.QueryRowContext(ctx,
		`INSERT INTO examples (id, name) VALUES (?, ?) RETURNING id, name, created_at`,
		ex.ID, ex.Name,
	).Scan(&ex.ID, &ex.Name, &ex.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert example: %w", err)
	}

	return ex, nil
}
