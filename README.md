## Lux

Lux contains a Go API server and a React frontend.

### Backend

```bash
make run
```

The API listens on `http://localhost:8080`. Health, readiness, and Swagger are
available at `/health`, `/ready`, and `/swagger/index.html`.

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
