# Pagination & latency: benchmark first, then pool-level caching

## What the code says today (the problem, precisely)

1. **Every page click is a full re-fan-out.** `Search()` builds `fetchRequest.Limit = page × limit`, calls all 7 providers, re-fuses the pool, and slices. The response cache keys on `page`, so page 2 of the same query is a guaranteed cache miss — you pay the slowest provider's latency (measured earlier this session: ~2.6–3.5s when upstreams are healthy, 25s when gutendex hangs) on *every* page navigation, and every page click hammers all upstreams again (the rate-limit risk).
2. **Pagination has a hard depth ceiling nobody advertised.** Providers cap their fetch: OL/LoC/DOAB at 100, IA/PG at 200. The frontend uses `limit=24` and allows `page≤20` — but page 5 needs 120 per provider, more than OL/LoC/DOAB can return. The pool runs dry around **page 4–5**; `has_more` honestly goes false, but the "page ≤ 20" API promise is fiction.
3. **Latency is bounded by the slowest provider** (fan-out is already parallel). Tail = gutendex's 25s hangs. Fixing the tail is a separate, riskier change (early-return/shorter budgets) — noted as follow-up, not in this batch.

## Approach chosen: pool-level cache keyed *without* page (A)

Rejected alternatives: true per-provider upstream pagination (breaks stable RRF ordering across pages, doesn't reduce fan-out); server-side cursor/session state (stateful, bigger, unnecessary); frontend-side paging of one big fetch (changes API semantics).

**Design:**
- `searchCache` entries become `{pool []SearchResult, providers []ProviderStatus, fetchLimit int, expiresAt int64}`, keyed by the current key **minus `page`**. `pool` is the fused, sorted, unsliced pool; `fetchLimit` records the `page×limit` depth the providers actually served.
- On a hit with `page×limit ≤ fetchLimit`: slice locally — page navigation becomes ~0 ms with zero upstream calls, ordering identical by construction. On a hit needing deeper data than the pool holds: treat as miss, re-fetch deeper, replace the entry (so deep links work).
- Cache only when ≥1 provider succeeded (unchanged rule). Statuses travel with the pool (frozen for TTL — already the documented behavior).
- Cap at 128 pool entries (bigger payloads than today's response cache), same TTL/eviction.
- Max page stays 20; the *effective* depth ceiling comes from provider caps and is honest via `has_more`. Document it in the README (~4 pages at limit=24).

**Plus one cheap observability add:** search-duration buckets (100ms → 25s) on `/metrics`, so the healthy-vs-gutendex-hangs tail is visible after launch.

## Steps

1. **Step 0 — live benchmark (your "test live first"):** build the server to `/tmp`, run on a scratch port with a scratch DB (nothing in the repo touched), measure and record: page-1 cold, page-2, page-3, page-1 repeat (cache hit), `/metrics` fan-out counts per page nav, and a probe of the effective depth ceiling. These become the before/after numbers.
2. **Rework `search_cache.go` + `Search()`** to the pool design above (search.go slicing moves after the cache check; `fetchRequest` still computed the same way on miss).
3. **Metrics buckets** in `internal/metrics` + wire in `books`.
4. **Tests:** page-nav-is-a-hit (page 1 → page 2 → provider called once), deeper-than-pool re-fetch, direct deep-link, metrics bucket counts; keep existing tests green (cache-key change may need small updates).
5. **Re-run the Step 0 benchmark** → before/after table (expect: page navs drop from seconds to ~0 ms, provider fan-outs per page nav from 7 to 0).
6. **README** pagination paragraph update (pool cache + depth ceiling), TODO-free docs as usual, commit(s), push.

Scope: `server/internal/books/{search,search_cache}.go`, `server/internal/metrics/metrics.go`, tests, README. No API/behavior changes visible to clients except pages arriving ~instantly.