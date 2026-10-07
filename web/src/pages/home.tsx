import { Loader2 } from "lucide-react"
import { useEffect, useRef, useState } from "react"
import { useSearchParams } from "react-router-dom"

import { searchBooks, type ProviderStatus, type SearchResponse } from "@/api/books"
import { BookCard } from "@/components/book-card"
import { FilterToolbar, type FilterState } from "@/components/filters"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  GENRES,
  genreLabel,
  providerLabel,
} from "@/lib/book-display"

type LoadedSearch = {
  key: string
  response?: SearchResponse
  error?: string
}

function filtersFromParams(searchParams: URLSearchParams): FilterState {
  return {
    language: searchParams.get("language") ?? "",
    minRating: searchParams.get("min_rating") ?? "",
    minYear: searchParams.get("min_year") ?? "",
    maxYear: searchParams.get("max_year") ?? "",
  }
}

function setOrDelete(params: URLSearchParams, key: string, value: string) {
  if (value) {
    params.set(key, value)
  } else {
    params.delete(key)
  }
}

function joinAnd(names: string[]) {
  if (names.length <= 1) return names.join("")
  if (names.length === 2) return `${names[0]} and ${names[1]}`
  return `${names.slice(0, -1).join(", ")} and ${names[names.length - 1]}`
}

// One quiet line under the results header: who answered, and a collapsed
// detail for who didn't, so a blocked catalog doesn't read as a broken site.
function ProviderLine({ statuses }: { statuses: ProviderStatus[] }) {
  if (statuses.length === 0) return null
  const served = statuses.filter((s) => s.status === "ok" || s.status === "partial")
  const failed = statuses.filter((s) => s.status === "error")
  const skipped = statuses.filter((s) => s.status === "skipped")

  return (
    <div className="flex flex-wrap items-center gap-x-3 gap-y-1 border-b border-border pb-3.5 font-sans text-[13px] text-muted-foreground">
      {served.length > 0 && (
        <span
          className="text-foreground"
          title={served.map((s) => `${providerLabel(s.provider)}: ${s.count ?? 0} books`).join(", ")}
        >
          <span className="mr-1.5 inline-block size-[7px] rounded-full bg-primary align-middle" />
          Results from{" "}
          {joinAnd(
            served.map((s) => providerLabel(s.provider) + (s.status === "partial" ? " (partial)" : "")),
          )}
        </span>
      )}
      {failed.length > 0 && (
        <details className="relative">
          <summary className="cursor-pointer list-none underline decoration-dotted underline-offset-[3px] hover:text-foreground">
            {failed.length} {failed.length === 1 ? "catalog" : "catalogs"} unavailable
          </summary>
          <div className="absolute left-0 top-6 z-30 w-80 rounded-lg border border-border bg-card p-3 shadow-[0_8px_22px_rgba(60,50,30,0.12)]">
            <p>{joinAnd(failed.map((s) => providerLabel(s.provider)))} couldn't return results.</p>
            {failed.some((s) => s.reason) && (
              <ul className="mt-1.5 space-y-0.5">
                {failed
                  .filter((s) => s.reason)
                  .map((s) => (
                    <li key={s.provider}>
                      {providerLabel(s.provider)}: {s.reason}
                    </li>
                  ))}
              </ul>
            )}
            <p className="mt-1.5">Available results are shown below.</p>
          </div>
        </details>
      )}
      {skipped.length > 0 && (
        <details className="relative">
          <summary className="cursor-pointer list-none underline decoration-dotted underline-offset-[3px] hover:text-foreground">
            {skipped.length} {skipped.length === 1 ? "catalog" : "catalogs"} skipped
          </summary>
          <div className="absolute left-0 top-6 z-30 w-80 rounded-lg border border-border bg-card p-3 shadow-[0_8px_22px_rgba(60,50,30,0.12)]">
            <ul className="space-y-0.5">
              {skipped
                .filter((s) => s.reason)
                .map((s) => (
                  <li key={s.provider}>
                    {providerLabel(s.provider)}: {s.reason}
                  </li>
                ))}
            </ul>
          </div>
        </details>
      )}
    </div>
  )
}

function fetchParams(searchParams: URLSearchParams) {
  return {
    language: searchParams.get("language") ?? undefined,
    minRating: searchParams.get("min_rating")
      ? Number(searchParams.get("min_rating"))
      : undefined,
    minYear: searchParams.get("min_year") ? Number(searchParams.get("min_year")) : undefined,
    maxYear: searchParams.get("max_year") ? Number(searchParams.get("max_year")) : undefined,
  }
}

export function HomePage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const topic = searchParams.get("topic") ?? ""
  const genre = searchParams.get("genre") ?? "non-fiction"
  const page = Number(searchParams.get("page")) || 1
  const requestKey = searchParams.toString()
  const searched = Boolean(topic) || searchParams.has("genre")

  // The URL is the single source of truth: filters are read straight from it
  // and every control mutates it, so paging, genre chips, and browser
  // back/forward always agree. Only the topic box is staged (it commits on
  // submit), and it re-syncs whenever the URL changes for a reason other than
  // this component's own update — back/forward navigation in particular.
  const filters = filtersFromParams(searchParams)
  const [topicInput, setTopicInput] = useState(topic)
  const lastPushedKey = useRef<string | null>(null)
  const [loaded, setLoaded] = useState<LoadedSearch>({ key: "" })
  const resultsRef = useRef<HTMLElement>(null)

  useEffect(() => {
    if (lastPushedKey.current !== null && requestKey === lastPushedKey.current) return
    setTopicInput(searchParams.get("topic") ?? "")
  }, [requestKey, searchParams])

  useEffect(() => {
    if (!searched) return
    const key = requestKey
    const controller = new AbortController()
    searchBooks(
      {
        topics: topic ? [topic] : [],
        genre,
        page,
        ...fetchParams(searchParams),
        limit: 24,
      },
      { signal: controller.signal },
    )
      .then((response) => {
        if (!controller.signal.aborted) setLoaded({ key, response })
      })
      .catch((err: Error) => {
        if (!controller.signal.aborted) setLoaded({ key, error: err.message })
      })
    return () => controller.abort()
  }, [searched, topic, genre, page, requestKey, searchParams])

  function pushParams(mutate: (params: URLSearchParams) => void) {
    const params = new URLSearchParams(searchParams)
    mutate(params)
    lastPushedKey.current = params.toString()
    setSearchParams(params)
  }

  // Committing a search replaces the topic and resets paging; the genre and
  // filters in the URL are kept so refining a topic preserves the view.
  function submitSearch() {
    pushParams((params) => {
      if (topicInput.trim()) {
        params.set("topic", topicInput.trim())
      } else {
        params.delete("topic")
      }
      params.delete("page")
    })
  }

  // A genre-chip click must always write the param: without it, a default-genre
  // browse produces an empty URL and `searched` never turns true.
  function browseGenre(nextGenre: string) {
    pushParams((params) => {
      params.set("genre", nextGenre)
      params.delete("page")
    })
  }

  // Filter edits apply immediately and jump back to page 1 of the refined view.
  function applyFilters(next: FilterState) {
    pushParams((params) => {
      setOrDelete(params, "language", next.language)
      setOrDelete(params, "min_rating", next.minRating)
      setOrDelete(params, "min_year", next.minYear)
      setOrDelete(params, "max_year", next.maxYear)
      params.delete("page")
    })
  }

  function changePage(next: number) {
    pushParams((params) => {
      if (next <= 1) {
        params.delete("page")
      } else {
        params.set("page", String(next))
      }
    })
    resultsRef.current?.scrollIntoView?.({ behavior: "smooth", block: "start" })
  }

  const current = loaded.key === requestKey ? loaded : undefined
  const loading = searched && !current
  const error = current?.error
  const response = current?.response
  const results = response?.results ?? []
  const providerStatuses = response?.providers ?? []

  // No federated total exists, so count what is loaded so far and let "+"
  // signal that more pages are available.
  const loadedSoFar = response ? (page - 1) * (response.limit ?? 24) + results.length : 0
  const scopeLabel = genreLabel(genre).replace(" (all)", "")
  const countNode =
    !loading && !error && results.length > 0 ? (
      <p className="mr-auto font-serif text-[19px]">
        <span className="font-semibold">
          {loadedSoFar.toLocaleString("en")}
          {response?.has_more ? "+" : ""}
        </span>{" "}
        {loadedSoFar === 1 ? "book" : "books"}{" "}
        <span className="font-sans text-[15px] text-muted-foreground">in {scopeLabel}</span>
      </p>
    ) : undefined

  const chipClass = (active: boolean) =>
    `rounded-full border px-[15px] py-1.5 font-sans text-sm transition-colors ${
      active
        ? "border-foreground bg-foreground font-medium text-background"
        : "border-border bg-card text-muted-foreground hover:bg-accent hover:text-accent-foreground"
    }`

  return (
    <div className="space-y-8">
      <section className="mx-auto max-w-3xl space-y-3 pt-6 text-center">
        <p className="font-sans text-[13px] font-semibold uppercase tracking-[0.18em] text-primary">
          Trafae · Free book discovery
        </p>
        <h1 className="font-serif text-4xl leading-[1.15] tracking-tight sm:text-[44px]">
          Find books on a topic across open book collections.
        </h1>
        <p className="mx-auto max-w-2xl text-[17px] text-muted-foreground">
          One search across Project Gutenberg, Open Library, Internet Archive, DOAB
          and more. Read or download where available; access terms vary by collection.
        </p>
      </section>

      <section aria-label="Book search" className="space-y-4">
        <form
          className="mx-auto flex max-w-2xl flex-col gap-3 sm:flex-row"
          onSubmit={(event) => {
            event.preventDefault()
            submitSearch()
          }}
        >
          <Input
            className="h-12 flex-1 rounded-l-lg rounded-r-none border-[1.5px] border-foreground bg-white font-serif text-[17px] placeholder:italic placeholder:text-stone-400"
            onChange={(event) => setTopicInput(event.target.value)}
            placeholder="Search a topic, like psychology or ancient Rome…"
            value={topicInput}
          />
          <Button className="h-12 rounded-l-none rounded-r-lg px-6 font-sans text-base" size="lg" type="submit">
            Search the stacks
          </Button>
        </form>

        <div aria-label="Genres" className="flex flex-wrap justify-center gap-2">
          {GENRES.map((item) => (
            <button
              className={chipClass(item === genre)}
              key={item}
              onClick={() => browseGenre(item)}
              type="button"
            >
              {genreLabel(item)}
            </button>
          ))}
        </div>

        {!searched && (
          <p className="text-center font-sans text-sm text-muted-foreground">
            Pick a topic above, or browse a genre. No search term needed.
          </p>
        )}
      </section>

      {searched && (
        <section aria-label="Results" className="scroll-mt-6 space-y-4" ref={resultsRef}>
          {!loading && !error && (
            <FilterToolbar count={countNode} filters={filters} onChange={applyFilters} />
          )}

          <ProviderLine statuses={providerStatuses} />

          {loading && (
            <div className="flex items-center gap-2 py-16 font-serif italic text-muted-foreground">
              <Loader2 className="size-4 animate-spin" /> Searching the stacks…
            </div>
          )}

          {!loading && error && (
            <div className="rounded-lg border border-destructive/40 bg-card p-5">
              <p className="font-semibold text-destructive">Search failed</p>
              <p className="mt-1 font-sans text-sm text-muted-foreground">{error}</p>
            </div>
          )}

          {!loading && !error && results.length === 0 && (
            <div className="rounded-lg border border-border bg-card p-8 text-center">
              <p className="font-semibold">
                {page > 1 ? "This page is past the end of the results" : "No books matched"}
              </p>
              <p className="mt-1 font-sans text-sm text-muted-foreground">
                {page > 1
                  ? "Step back to the last page with Previous."
                  : "Try a broader topic, another genre, or fewer filters."}
              </p>
            </div>
          )}

          {!loading && !error && results.length > 0 && (
            <div className="grid gap-[22px] py-5 md:grid-cols-2 xl:grid-cols-3">
              {results.map((result, index) => {
                const { book } = result
                return (
                  <BookCard
                    key={
                      book.source.id
                        ? `${book.source.provider}:${book.source.id}`
                        : `${book.source.provider}:${book.title}:${book.year ?? ""}:${index}`
                    }
                    result={result}
                  />
                )
              })}
            </div>
          )}

          {!loading && !error && (results.length > 0 || page > 1) && (
            <nav aria-label="Result pages" className="flex items-center justify-center gap-4 border-t border-border pt-5">
              <Button
                disabled={page <= 1}
                onClick={() => changePage(page - 1)}
                size="sm"
                variant="outline"
              >
                ← Previous
              </Button>
              <span className="font-serif text-sm text-muted-foreground">Page {page}</span>
              <Button
                disabled={!response?.has_more}
                onClick={() => changePage(page + 1)}
                size="sm"
                variant="outline"
              >
                Next →
              </Button>
            </nav>
          )}
        </section>
      )}
    </div>
  )
}
