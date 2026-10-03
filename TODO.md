# Tech Debt & Backlog

Systematic audit of the codebase from **2026-10-03** (after the topic/genre browse +
pagination work). Each item links to its GitHub issue; checkboxes track completion.
Legend: 🔴 high · 🟡 medium · ⚪ low.

## Bugs

- [x] 🟡 **Provider chip hides the count when a catalog returns 0 books** — `count` had
  `omitempty`, so a successful zero-result search rendered "undefined" in the chip tooltip.
  Fixed: `count` is always serialized and the chip falls back to `?? 0`; also renders the
  new `partial` status. (`server/internal/books/types.go:105`, `web/src/pages/home.tsx`) [#19](https://github.com/Chandra179/trafae/issues/19)
- [x] 🔴 **Gutendex drops every topic after the first** — the page-walk loop condition uses
  the accumulator shared across all topic terms, so term 1 fills it and terms 2..8 never
  execute. (`server/internal/providers/gutendex.go:44`) [#3](https://github.com/Chandra179/trafae/issues/3)
- [x] 🔴 **Shared http.Client timeout caps per-provider timeouts** — client `Timeout: 18s`
  includes body read, so the configured 25–30s provider budgets can never be reached.
  Fixed: no client-level timeout; per-provider context deadlines govern, and the now-dead
  `http_timeout_in_second` knob was removed from config. (`server/server.go:82`, `config.go:89-91`, `provider_utils.go:183`) [#4](https://github.com/Chandra179/trafae/issues/4)
- [x] 🟡 **One failed upstream request discards all of a provider's results** — every
  sequential provider returns `nil, err` mid-loop instead of returning partial results.
  Fixed: per-term errors are collected (`searchTerms`/`termFailure`); a provider now returns
  its partial books, and the fusion layer reports a new `partial` provider status.
  (`open_library.go:66`, `doab.go:57`, `library_of_congress.go:48`, `internet_archive.go:50`,
  `gutendex.go:61`, `wikidata.go:44`) [#6](https://github.com/Chandra179/trafae/issues/6)
- [ ] 🟡 **Language filter declared but never sent upstream** by Open Library, DOAB, Internet
  Archive and Wikidata — capped fetch windows can saturate with wrong-language hits, yielding
  zero results despite upstream matches. [#7](https://github.com/Chandra179/trafae/issues/7)
- [x] 🟡 **Schema lives in two places; server never migrates on boot** — no goose wiring in Go
  code, and PG tables are created twice (`ensureSchema` vs migration `00002`), which can drift.
  Fixed: embedded goose migrations are applied on boot (`store.Migrate`, also exposed as
  `server/cmd/migrate`); `ensureSchema` and its mutex/flag are deleted; migration `00002` is
  idempotent so pre-goose databases migrate cleanly; CI and `make migrate-up` run the same
  code path instead of `goose@latest`.
  (`store/migrate.go`, `server.go`, `project_gutenberg.go`, `cmd/migrate/`) [#9](https://github.com/Chandra179/trafae/issues/9)
- [ ] 🟡 **Swagger UI route is dead** — `/swagger/*any` is mounted but no swag docs package is
  generated or imported; README advertises it. (`router/router.go:55`) [#10](https://github.com/Chandra179/trafae/issues/10)
- [x] 🟡 **Project Gutenberg feed refresh races the 18s client timeout** — full gzipped-CSV
  download streamed through the shared client. Fixed: the feed uses a dedicated client with
  a 10-minute budget. (`project_gutenberg.go:129`) [#8](https://github.com/Chandra179/trafae/issues/8)
- [x] 🟡 **"Non-fiction (all)" chip can never start a search** — `runSearch` omits the default
  genre from the URL while `searched` requires topic or genre param.
  (`web/src/pages/home.tsx:199,162`) [#12](https://github.com/Chandra179/trafae/issues/12)
- [x] 🟡 **Stale `?page` past the last page dead-ends** — empty results render without
  pagination controls, so there is no way back to page 1. (`web/src/pages/home.tsx:317-356`)
  [#13](https://github.com/Chandra179/trafae/issues/13)
- [ ] 🟡 **Search inputs don't re-sync with URL on back/forward nav; staged filter edits are
  silently dropped by paging.** (`web/src/pages/home.tsx:164-171,280,208`)
  [#14](https://github.com/Chandra179/trafae/issues/14)
- [ ] 🟡 **Genre matching heuristic is loose and duplicated** — bidirectional substring matching
  over-matches; the non-fiction keyword list is hardcoded in two places.
  (`search.go` `genreMatches`, `project_gutenberg.go:42-43`) [#16](https://github.com/Chandra179/trafae/issues/16)

## Performance

- [x] 🔴 **Per-topic upstream requests run sequentially inside one provider call** — worst case
  8 topics ⇒ Open Library alone issues 8 sequential requests plus ~7s of rate-limit sleeps,
  all inside a single provider context; gutendex walks up to 8 pages per term.
  Fixed: per-term requests fan out with a bounded concurrency of 4 (`searchTerms`); Open
  Library's rate limiter still spaces request starts.
  (`open_library.go:43-96`, `doab.go:32-80`, `library_of_congress.go:32-65`,
  `internet_archive.go:31-73`, `gutendex.go:32-84`, `wikidata.go:43-51`)
  [#5](https://github.com/Chandra179/trafae/issues/5)
- [x] 🟡 **Cover images load eagerly** — 24 remote fetches per page; add
  `loading="lazy" decoding="async"`. (`web/src/components/book-card.tsx:41-46`)
  [#18](https://github.com/Chandra179/trafae/issues/18)

## Testing & CI

- [ ] 🟡 **Zero frontend tests** — no vitest, no test script; CI runs only lint/typecheck/build
  for `web/`. The URL-state machine in `home.tsx` is the most intricate logic in the app.
  [#15](https://github.com/Chandra179/trafae/issues/15)
- [ ] 🟡 **Unpinned tool versions** — golangci-lint `version: v2` (floating), goose `@latest`,
  and three different Go version references (`go.mod` 1.26.5 vs CI 1.27.0 vs Containerfile
  `golang:1.27`). [#11](https://github.com/Chandra179/trafae/issues/11)

## Dead code & cleanup

- [ ] ⚪ **Backend**: Badger opened but never stores anything (whole dep + `badger.dir` config
  pointless); gRPC interceptor never registered (sole reason for the grpc/protobuf deps);
  unused generated mocks in `server/mocks/`; unused provider base-URL config seams; empty
  scaffold files (`AGENTS.md`, `lefthook.yml`, `internal/constant/`).
  [#17](https://github.com/Chandra179/trafae/issues/17)
- [ ] ⚪ **Frontend**: unused `badge.tsx`, `api/health.ts`, `api/examples.ts`,
  `providerSummary`, `lib/utils.ts` (dead `cn` split — npm `cn` package vs `@/lib/utils`,
  making `clsx` + `tailwind-merge` dead deps too); `radix-ui` Slot and `tw-animate-css`
  effectively unused; unused `CardTitle`/`CardDescription`/`CardAction` exports.
  [#17](https://github.com/Chandra179/trafae/issues/17)
- [x] ⚪ **Small hardening batch**: cap string param lengths; fix misleading `limit` error
  text; add `page` to the dev log allowlist; bound year inputs; description line-clamp and
  title `break-words`; result-key collision fallback.
  [#18](https://github.com/Chandra179/trafae/issues/18)

## Docs

- [ ] ⚪ `server/internal/README.md` describes a Mockery-through-`export_test.go` pattern that
  does not exist, and prescribes exported `Service` interfaces that have zero consumers.
- [ ] ⚪ Document provider network limitations (below) in the README once the provider set
  stabilizes.

## Known limitations (not bugs — environmental/accepted tradeoffs)

- **DOAB returns HTTP 403 from this deployment network** ("Your address is not allowed to
  access this API") — the provider was migrated to DOAB's current DSpace 7 discover API, but
  their API gate blocks some client IPs. Re-test from a different network before assuming a
  code problem.
- **loc.gov returns HTTP 403 (Cloudflare "Just a moment" challenge)** for datacenter IPs —
  browser-like User-Agents do not help. May work from other networks.
- **Gutendex occasionally times out** from some networks (observed once in testing); it has
  a 25s configured budget and recovers on retry.
- **Deep pagination re-fetches and re-fuses the whole result pool on every page request** —
  this is the price of stateless, stable RRF ordering. If it becomes hot, add a short-TTL
  cache keyed by the normalized query.
- **Open Library sometimes serves blank/white cover images** that load successfully, so the
  `onError` gradient fallback cannot detect them.

## Verified clean during the audit

WAL/busy-timeout DSN pragmas, `ensureSchema` retry/mutex logic, provider fan-out channel
discipline, Open Library cache locking, `safeNextURL` SSRF guard, IA query escaping, root
README accuracy (params, endpoints, limits), Vite proxy config, node/engines alignment,
`docs/discovery/` freshness.
