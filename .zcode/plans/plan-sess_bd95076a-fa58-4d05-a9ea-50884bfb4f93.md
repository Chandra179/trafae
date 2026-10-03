# Topic/genre-first search + pagination + navbar removal

## Answers to your questions (verified against code + upstream API docs)

1. **New endpoint or reuse?** Reuse `GET /books/search`. There is no title field anywhere today — the single search term is already treated as a topic that also matches titles/subjects server-side (`matchesRequest`, search.go:206). Topic-less requests are already valid on the backend; only the frontend requires a topic. No new endpoint, no new param needed.
2. **Do providers support pagination?** All 7 can paginate upstream, but our backend exposes none today (it fuses results and slices at `limit`; no `page`/`offset` anywhere):
   | Provider | Upstream paging | Used today |
   |---|---|---|
   | open_library | `page` + `limit` (~100/page max) | no |
   | gutendex | `page` / `next` links | partially (internal 3-page walk) |
   | internet_archive | `page` + `rows` | no (`page=1` hardcoded) |
   | doab | DSpace 7 `page` (0-based) + `size` | no (dead legacy endpoint, no size sent) |
   | library_of_congress | `sp` (page) + `c` (count) | no |
   | wikidata | SPARQL `OFFSET` | no |
   | project_gutenberg | SQL `LIMIT/OFFSET` (local DB) | no |
3. **Topic vs genre support?** All 7 providers support topic/keyword search. **None send genre upstream** — genre is a Go post-filter for every provider; only Wikidata and Project Gutenberg reuse the genre string as fallback search terms when no topics are given.

## Backend (all in `server/`)

**A. Pagination on `/books/search` — stateless "fetch deeper, then slice"** (correct fused RRF ordering across pages, no server-side session state):
1. `handler.go`: add `page` query param (1-based, default 1, validated 1–20); `types.go`: `SearchRequest.Page`, `SearchResponse.Page` + `HasMore bool`.
2. `search.go`: build the provider-facing request copy with `Limit = page * limit` (bounded) so each provider returns enough to fill deeper pages; raise the per-provider fetch cap in `withLimit` call sites from 100 → 200 (smoke-test Open Library at 200; fall back to 100 for OL if it rejects). After RRF fusion + sort, slice `pool[(page-1)*limit : page*limit]`; `HasMore = len(pool) > page*limit`.
3. Provider adjustments so deeper fetches actually reach deeper upstream: gutendex walk gets an adaptive page count (`ceil(target/pageSize)+1`, hard cap 6); IA keeps one request with bigger `rows` (drop nothing — `page` stays 1); OL `limit=min(fetchLimit, its cap)`; LoC `c`; wikidata SPARQL `LIMIT`; PG SQL `LIMIT`. No upstream page params needed anywhere except gutendex's existing walk — deep pages are served by bigger single batches.
4. Tests: page validation bounds, slice/`has_more` correctness with fake providers, updated provider param pins.

**B. Genre-only browse (empty search box + genre chip)**: in `Search()`, when topics are empty and genre is set (≠ default "non-fiction"), seed the provider-facing topics with the genre string so all providers keyword-search it (Wikidata/PG already do this internally; this extends it to all seven). Default non-fiction keeps current behavior.

**C. Provider repairs**:
- **DOAB migration to DSpace 7**: legacy `/rest/search` is dead (ECONNRESET from three independent fetchers + failed in live UI test while OL/IA succeeded). Rewrite `doab.go` against `https://directory.doabooks.org/server/api/discover/search/objects` with `query=<term>&page=<0-based>&size=<n>`, parse `_embedded.searchResult._embedded.objects[].metadata` (dc.title / dc.contributor.author / dc.date.issued / dc.subject / dc.identifier.uri); access URL = DOAB handle page; license = "Open Access". Exact metadata keys verified against a live payload during implementation.
- **User-Agent header**: add a proper `User-Agent` (e.g. `lux/0.1 (+https://github.com/Chandra179/trafae)`) on all provider requests in `provider_utils.go` — Wikidata/Wikimedia endpoints commonly 403 default Go clients; likely also fixes Gutendex/LoC failures. Diagnose each failing provider from `ProviderStatus.reason` and verify chips go green in the live smoke test.

## Frontend (`web/`)

**D. Remove navbar**: strip the `<header>` from `app-shell.tsx` (keep the max-width wrapper + `Outlet`; `App.tsx` untouched). Brand moves into the hero kicker ("Lux · Free book discovery") so the name isn't lost.

**E. Topic-optional search flow** in `home.tsx`:
- Search fires when topic **or** genre is present (remove `if (!topic) return`); empty box + genre chip = browse by genre. Empty box + no genre shows a "pick a topic or browse by genre" hint instead of nothing.
- Genre chips, filters, URL-synced state all unchanged otherwise.

**F. Prev/Next pagination UI** (your choice): `page` synced to URL (`?topic=psychology&page=2`), Previous/Next buttons + "Page N" indicator, Prev disabled on page 1, Next disabled when `!has_more` or loading; page resets to 1 on every new search; scroll back to results top on page change. `api/books.ts`: add `page?` to `BookSearchParams`, `page`/`has_more` to `SearchResponse`.

## Docs & verification

- README: document `page`/`has_more`, genre-only browse, DOAB's new API.
- `go test ./...` + golangci-lint; `npm run lint` + `typecheck` + `build` in web/.
- Live smoke on dev servers: "psychology" page 1 → 2 (stable ordering, no dupes), genre-only "history" browse, provider chips green where the network allows, navbar gone.