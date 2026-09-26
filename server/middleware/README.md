# Middleware

Gin middleware used by the HTTP server. Registered in order via `r.Use(...)`:

```
gin.CustomRecovery → RequestID → RequestBodyLimit → RequestLog → handler
```

## Files

| File | Kind | Description |
|------|------|-------------|
| `dependencies.go` | infra | Holds the `*zap.Logger` used by stateful middleware |
| `request_id.go` | middleware | Reads/reuses `X-Request-ID` header, generating a random ID if absent. Stores the ID in context, echoes it in the response. Also exports `RequestIDUnaryInterceptor` for gRPC and `GetRequestID(ctx)` for handlers. |
| `request_log.go` | middleware | Logs one canonical line per normal request: method, route pattern, status, duration, request ID, and optional allowlisted query params. Level tracks response status (Info for 2xx/3xx, Warn for 4xx, Error for 5xx); the last error attached via `c.Error(err)` is included for 4xx/5xx. Skips configured paths. Neither request nor response bodies are logged (see comment in file). Full rationale and log-level/stacktrace behavior: [`docs/logging.md`](../docs/logging.md). |
| `RequestBodyLimit` | middleware | Wraps request bodies with `http.MaxBytesReader` to reject oversized payloads before handlers decode them. |

## Why no body logging?

After JSON decoding, the body seen in middleware is not what the client sent
(whitespace stripped, keys reordered, unknown fields dropped). Logging it
provides no debugging value and risks PII leakage — the same reasoning applies
to response bodies. Query logging is disabled in production and uses an
allowlist plus sensitive-key redaction when enabled. See
[`docs/logging.md`](../docs/logging.md) for the full policy.

## Why no RealIP middleware?

Gin's `engine.SetTrustedProxies()` + `c.ClientIP()` handle `X-Forwarded-For` / `X-Real-IP` with CIDR-based trust filtering. No custom middleware needed.

## Why no Recovery middleware?

`gin.CustomRecovery` handles panics and writes a 500 response. The recovery callback uses `zap` so stack traces go to structured logs, not stdout.

## Why no validation middleware?

`c.ShouldBindJSON(&req)` + `binding` struct tags replace the old `DecodeAndValidate[T]` helper. Keep validation logic in handlers, not middleware.
