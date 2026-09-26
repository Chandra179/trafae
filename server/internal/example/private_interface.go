package example

import "context"

type store interface {
	CreateExample(ctx context.Context, name string) (*Example, error)
}

var _ store = (*storeDependencies)(nil)
