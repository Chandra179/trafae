package example

import (
	"database/sql"

	"go.uber.org/zap"
)

type DependenciesConfig struct {
	Logger *zap.Logger
	DB     *sql.DB
}

type dependencies struct {
	logger *zap.Logger
	store  store
}

type storeDependencies struct {
	db *sql.DB
}

var _ Service = (*dependencies)(nil)

func NewDependencies(deps *DependenciesConfig) *dependencies {
	return newDependencies(deps.Logger, &storeDependencies{db: deps.DB})
}

func newDependencies(logger *zap.Logger, store store) *dependencies {
	return &dependencies{logger: logger, store: store}
}
