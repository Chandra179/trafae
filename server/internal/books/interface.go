package books

import "context"

type Service interface {
	Search(context.Context, SearchRequest) (SearchResponse, error)
	Capabilities() []ProviderCapability
}

var _ Service = (*dependencies)(nil)
