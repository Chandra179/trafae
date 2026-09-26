package example

import (
	"context"
)

type Service interface {
	CreateExample(ctx context.Context, name string) (*Example, error)
}

var _ Service = (*dependencies)(nil)
