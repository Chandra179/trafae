## Lux

Lux is a book discovery service: it searches multiple open book catalogs at
once, deduplicates the results, and fuses them into a single ranked list. It
contains a Go API server and a React frontend.

### Book search

`GET /books/search` fans a query out to every configured provider in parallel
and merges the responses with Reciprocal Rank Fusion. Duplicate books across
providers are collapsed into one result that keeps the per-source metrics.

Supported providers:

| Provider | Popularity | Ratings | Notes |
|----------|------------|---------|-------|
| Open Library | — | 0–5 | Rate limited; set `open_library_contact_email` for higher limits |
| DOAB | — | — | Scholarly open-access books |
| Gutendex | Downloads | — | Project Gutenberg via Gutendex |
| Internet Archive | Downloads | 0–5 | |
| Library of Congress | — | — | Digital collections |
| Wikidata | — | — | Community-supplied metadata |
| Project Gutenberg | Downloads | — | Official catalog synced daily into SQLite |

`GET /books/providers` lists each provider with the filters it supports, so
clients can adapt queries to what will actually be served.

Example:

```bash
curl "http://localhost:8080/books/search?topic=biology&genre=non-fiction&min_rating=4&limit=10"
```

### Backend

```bash
make run
```

The API listens on `http://localhost:8080`. Health, readiness, and Swagger are
available at `/health`, `/ready`, and `/swagger/index.html`.

On startup the server syncs the official Project Gutenberg catalog (a gzipped
CSV) into SQLite and refreshes it every 24 hours. Searches against it run
locally; the other providers are called over HTTP.

Schema changes are applied with goose (the server does not migrate on boot):

```bash
make migrate-up
```

### Configuration

Configuration lives in `server/config/config_<APP_ENVIRONMENT>.yaml` (`dev` by
default); `SQLITE_DSN` and `BADGER_DIR` can be overridden by environment
variables. The books search knobs:

```yaml
books:
  default_genre: "non-fiction"   # applied when the request omits genre
  default_limit: 20              # applied when the request omits limit (max 50)

providers:
  http_timeout_in_second: 18     # shared HTTP client timeout for provider APIs
  search_timeout_in_second: 30   # default per-provider context timeout
  search_timeouts_in_second:     # per-provider overrides of that timeout
    open_library: 20
    doab: 15
    gutendex: 25
    internet_archive: 25
    library_of_congress: 20
    wikidata: 30
    project_gutenberg: 10
```

SQLite is opened in WAL mode with a 10s busy timeout so reads keep serving
while the catalog refresh writes.

### Frontend

Install dependencies and start Vite:

```bash
make web-install
make web-dev
```

The frontend listens on `http://localhost:5173` and proxies `/api/*` to the Go
server. The API client can be pointed at another server with
`web/.env.local`:

```bash
VITE_API_BASE_URL=/api
```

Frontend checks:

```bash
make web-lint
make web-typecheck
make web-build
```

### Checks

```bash
make test   # go test -short -race ./...
make lint   # golangci-lint run ./...
make fmt    # gofmt
```
