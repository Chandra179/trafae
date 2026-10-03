# Modules

Each module is a Go package under `internal/<name>/`. A module owns its domain
logic, application behavior, transport, and dependency wiring.

## Structure

| File | Purpose |
|------|---------|
| `dependencies.go` | Exported `DependenciesConfig` struct callers fill in; unexported `dependencies` struct holding the module's wired deps (logger, stores, providers, metrics); `NewDependencies(*DependenciesConfig) *dependencies` constructor that applies defaults so a zero-value config still works. HTTP handlers are methods on `*dependencies`. |
| `types.go` | Domain types and the JSON wire types for handlers |
| `handler.go` | HTTP entrypoints: query/body binding, validation, status-code mapping |
| `<feature>.go` | One file per operation (e.g. `search.go`, `search_cache.go`) holding the `*dependencies` methods that implement it |
| `<feature>_test.go` | Package-internal tests using hand-written stubs and fakes (see `stubProvider` in `internal/books`, `responseClient` in `internal/providers`). No mock generator is used. |

There are no exported `Service` interfaces, no `interface.go`, and no
`export_test.go` indirection: packages keep their internals private, and tests
inside the package call `NewDependencies` with stubs directly.

## Cross-module communication

The consuming package owns the interface it needs. `internal/providers`
implements `books.Provider`, and that interface lives in
`internal/books/types.go` because books is the consumer. Modules never import
each other's unexported types. `server/server.go` is the only place that
constructs concrete dependencies and wires them together; it hands finished
handler methods to `router/` for route registration.
