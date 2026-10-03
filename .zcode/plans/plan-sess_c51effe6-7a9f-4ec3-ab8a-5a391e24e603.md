# Refactor: stdlib algorithms over hand-rolled code, flat logic, dead nil guards

All changes verified against current code; behavior preserved except one latent bug fixed (noted). Files: `server/internal/books/{search,search_cache,dependencies}.go`, `server/internal/books/handler.go`, `server/middleware/request_id.go`, `server/internal/providers/provider_utils.go`.

## 1. Better algorithms

- **`languageMatches`** (books/search.go): replace the per-call 20-key map literal + suffix checks with a package-level `languageCanonical` map (variant → canonical code) and separator-token comparison. Fixes latent bug: compound values like `"ger/fre"` never matched their first token before (suffix-only check). Add a direct unit test pinning: `["fre"]↔fr`, `["eng/ger"]↔de` (first token now matches), `["English"]↔en`, unknown passthrough `["sv"]↔sv`, non-match `["fre"]↔en`.
- **One language knowledge source**: export `books.CanonicalLanguageCode(value)` (the canonical map); `gutendexLanguage` in provider_utils.go becomes a delegate to it (its 14-pair scan table deleted); `marcLanguage`/`locLanguage`/`wikidataLanguage` keep their shape but build on it. `gutendex.go` call site unchanged.
- **`containsPhrase`** (books/search.go): 15-line token-sequence loop → space-padded `strings.Contains` (exact whole-word match on normalized text — the existing genre tests pin prehistory/cartography/nonfiction behavior).
- **`generateRequestID`** (middleware/request_id.go): manual `crypto/rand` 16-byte hex → `uuid.NewString()` (uuid already a direct dep; format passes `validRequestID`). Drop `crypto/rand`/`encoding/hex` imports.

## 2. Flatten nested logic

- **`genreMatches`**: three nested loops → flat guard structure with one new helper `containsAnyPhrase(haystacks, phrases)`. Fiction-exclusion semantics preserved exactly (still substring-based, per current behavior).
- **`matchesRequest`** topic check: `found`-flag loop → `slices.ContainsFunc` (one guard, no mutable flag).
- **`HandleSearch`**: `allSkipped` flag loop → `!slices.ContainsFunc(...)` for the 422 branch.

## 3. Remove redundant nil checks

- **`searchCache.get/put`**: drop `c == nil` (constructor always builds the cache; `ttl <= 0` disable stays).
- **books `dependencies`**: drop `if provider != nil` in `Capabilities()` and `if provider == nil { continue }` in the byID loop — construction guarantees non-nil entries.

Kept on purpose: `Metrics` nil-receiver guards (documented API — Metrics may be nil), constructor-defaulting nils (e.g. Gutenberg feed client).

## 4. Verify

- Full `go test -short -race ./...` green (existing genre/language/cache tests are the regression net, plus the new languageMatches test).
- `go vet` + golangci-lint identical to the 33-issue baseline.
- Web untouched. Single commit, pushed.