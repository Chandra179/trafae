import { Loader2 } from "lucide-react"
import { useEffect, useRef, useState } from "react"
import { useSearchParams } from "react-router-dom"

import { searchBooks, type ProviderStatus, type SearchResponse } from "@/api/books"
import { BookCard } from "@/components/book-card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  GENRES,
  genreLabel,
  providerLabel,
} from "@/lib/book-display"

const LANGUAGES = ["en", "fr", "de", "es", "it", "pt"] as const

const LANGUAGE_LABELS: Record<string, string> = {
  en: "English",
  fr: "French",
  de: "German",
  es: "Spanish",
  it: "Italian",
  pt: "Portuguese",
}

const RATINGS = [
  { value: undefined, label: "Any rating" },
  { value: 3.5, label: "3.5+" },
  { value: 4, label: "4.0+" },
  { value: 4.5, label: "4.5+" },
] as const

type FilterState = {
  language: string
  minRating: string
  minYear: string
  maxYear: string
}

const emptyFilters: FilterState = { language: "", minRating: "", minYear: "", maxYear: "" }

type LoadedSearch = {
  key: string
  response?: SearchResponse
  error?: string
}

function ProviderChip({ status }: { status: ProviderStatus }) {
  const label = providerLabel(status.provider)
  if (status.status === "ok") {
    return (
      <span
        className="inline-flex items-center gap-1.5 rounded border border-border bg-card px-2.5 py-1 font-sans text-xs text-foreground"
        title={`${label} returned ${status.count} books`}
      >
        <span className="size-[7px] rounded-full bg-primary" />
        {label} <span className="font-semibold">{status.count}</span>
      </span>
    )
  }
  if (status.status === "skipped") {
    return (
      <span
        className="inline-flex items-center gap-1.5 rounded border border-border bg-card px-2.5 py-1 font-sans text-xs text-muted-foreground"
        title={status.reason}
      >
        <span className="size-[7px] rounded-full bg-[#c9b98a]" />
        {label} skipped
      </span>
    )
  }
  return (
    <span
      className="inline-flex items-center gap-1.5 rounded border border-destructive/40 bg-card px-2.5 py-1 font-sans text-xs text-destructive"
      title={status.reason}
    >
      <span className="size-[7px] rounded-full bg-destructive" />
      {label} failed
    </span>
  )
}

function Filters({
  filters,
  onChange,
}: {
  filters: FilterState
  onChange: (filters: FilterState) => void
}) {
  const selectClass =
    "h-9 rounded-md border border-input bg-white px-2.5 font-sans text-sm text-foreground shadow-xs"
  const inputClass =
    "h-9 w-24 rounded-md border border-input bg-white px-2.5 font-sans text-sm shadow-xs placeholder:text-muted-foreground/60"

  return (
    <div className="flex flex-wrap items-center gap-2.5">
      <select
        aria-label="Language"
        className={selectClass}
        onChange={(event) => onChange({ ...filters, language: event.target.value })}
        value={filters.language}
      >
        <option value="">Any language</option>
        {LANGUAGES.map((code) => (
          <option key={code} value={code}>
            {LANGUAGE_LABELS[code]}
          </option>
        ))}
      </select>
      <select
        aria-label="Minimum rating"
        className={selectClass}
        onChange={(event) => onChange({ ...filters, minRating: event.target.value })}
        value={filters.minRating}
      >
        {RATINGS.map((rating) => (
          <option key={rating.label} value={rating.value ?? ""}>
            {rating.label}
          </option>
        ))}
      </select>
      <div className="flex items-center gap-1.5">
        <input
          aria-label="Year from"
          className={inputClass}
          onChange={(event) => onChange({ ...filters, minYear: event.target.value })}
          placeholder="From year"
          type="number"
          value={filters.minYear}
        />
        <span className="text-sm text-muted-foreground">–</span>
        <input
          aria-label="Year to"
          className={inputClass}
          onChange={(event) => onChange({ ...filters, maxYear: event.target.value })}
          placeholder="To year"
          type="number"
          value={filters.maxYear}
        />
      </div>
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

  const [topicInput, setTopicInput] = useState(topic)
  const [filters, setFilters] = useState<FilterState>({
    ...emptyFilters,
    language: searchParams.get("language") ?? "",
    minRating: searchParams.get("min_rating") ?? "",
    minYear: searchParams.get("min_year") ?? "",
    maxYear: searchParams.get("max_year") ?? "",
  })
  const [loaded, setLoaded] = useState<LoadedSearch>({ key: "" })
  const resultsRef = useRef<HTMLElement>(null)

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
      .then((response) => setLoaded({ key, response }))
      .catch((err: Error) => {
        if (!controller.signal.aborted) setLoaded({ key, error: err.message })
      })
    return () => controller.abort()
  }, [searched, topic, genre, page, requestKey, searchParams])

  function runSearch(nextTopic: string, nextGenre: string, nextFilters: FilterState) {
    const params = new URLSearchParams()
    if (nextTopic.trim()) params.set("topic", nextTopic.trim())
    if (nextGenre !== "non-fiction") params.set("genre", nextGenre)
    if (nextFilters.language) params.set("language", nextFilters.language)
    if (nextFilters.minRating) params.set("min_rating", nextFilters.minRating)
    if (nextFilters.minYear) params.set("min_year", nextFilters.minYear)
    if (nextFilters.maxYear) params.set("max_year", nextFilters.maxYear)
    setSearchParams(params)
  }

  function changePage(next: number) {
    const params = new URLSearchParams(searchParams)
    if (next <= 1) {
      params.delete("page")
    } else {
      params.set("page", String(next))
    }
    setSearchParams(params)
    resultsRef.current?.scrollIntoView({ behavior: "smooth", block: "start" })
  }

  const current = loaded.key === requestKey ? loaded : undefined
  const loading = searched && !current
  const error = current?.error
  const response = current?.response
  const results = response?.results ?? []
  const providerStatuses = response?.providers ?? []

  const chipClass = (active: boolean) =>
    `rounded-full border px-[15px] py-1.5 font-sans text-sm transition-colors ${
      active
        ? "border-foreground bg-foreground font-semibold text-background"
        : "border-border bg-card text-muted-foreground hover:bg-accent hover:text-accent-foreground"
    }`

  return (
    <div className="space-y-8">
      <section className="mx-auto max-w-3xl space-y-3 pt-6 text-center">
        <p className="font-sans text-[13px] font-semibold uppercase tracking-[0.18em] text-primary">
          Lux · Free book discovery
        </p>
        <h1 className="font-serif text-4xl font-semibold leading-[1.15] tracking-tight sm:text-[44px]">
          The best books on any topic — that you can start reading free, today.
        </h1>
        <p className="mx-auto max-w-2xl text-[17px] text-muted-foreground">
          One search across Project Gutenberg, Open Library, Internet Archive, DOAB
          and more. Legally free, no waitlists, straight to the book.
        </p>
      </section>

      <section aria-label="Book search" className="space-y-4">
        <form
          className="mx-auto flex max-w-2xl flex-col gap-3 sm:flex-row"
          onSubmit={(event) => {
            event.preventDefault()
            runSearch(topicInput, genre, filters)
          }}
        >
          <Input
            className="h-12 flex-1 rounded-l-lg rounded-r-none border-[1.5px] border-foreground bg-white font-serif text-[17px] placeholder:italic placeholder:text-stone-400"
            onChange={(event) => setTopicInput(event.target.value)}
            placeholder="Search a topic — psychology, ancient Rome, climate…"
            value={topicInput}
          />
          <Button className="h-12 rounded-l-none rounded-r-lg px-6 font-sans text-base font-semibold" size="lg" type="submit">
            Search the stacks
          </Button>
        </form>

        <div aria-label="Genres" className="flex flex-wrap justify-center gap-2">
          {GENRES.map((item) => (
            <button
              className={chipClass(item === genre)}
              key={item}
              onClick={() => runSearch(topicInput, item, filters)}
              type="button"
            >
              {genreLabel(item)}
            </button>
          ))}
        </div>

        <div className="flex justify-center">
          <Filters filters={filters} onChange={(next) => setFilters(next)} />
        </div>

        {!searched && (
          <p className="text-center font-sans text-sm text-muted-foreground">
            Pick a topic above, or browse a genre — no search term needed.
          </p>
        )}
      </section>

      {searched && (
        <section aria-label="Results" className="scroll-mt-6 space-y-4" ref={resultsRef}>
          {providerStatuses.length > 0 && (
            <div className="flex flex-wrap items-center gap-2 border-b border-border pb-3.5">
              <span className="mr-1 font-sans text-[13px] text-muted-foreground">
                Searched {providerStatuses.filter((p) => p.status === "ok").length} of{" "}
                {providerStatuses.length} catalogs:
              </span>
              {providerStatuses.map((status) => (
                <ProviderChip key={status.provider} status={status} />
              ))}
            </div>
          )}

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
              <p className="font-semibold">No books matched</p>
              <p className="mt-1 font-sans text-sm text-muted-foreground">
                Try a broader topic, another genre, or fewer filters.
              </p>
            </div>
          )}

          {!loading && !error && results.length > 0 && (
            <>
              <div className="grid gap-[22px] py-5 md:grid-cols-2 xl:grid-cols-3">
                {results.map((result) => (
                  <BookCard
                    key={`${result.book.source.provider}:${result.book.source.id ?? result.book.title}`}
                    result={result}
                  />
                ))}
              </div>
              <nav aria-label="Result pages" className="flex items-center justify-center gap-4 border-t border-border pt-5">
                <Button
                  disabled={page <= 1 || loading}
                  onClick={() => changePage(page - 1)}
                  size="sm"
                  variant="outline"
                >
                  ← Previous
                </Button>
                <span className="font-serif text-sm text-muted-foreground">Page {page}</span>
                <Button
                  disabled={!response?.has_more || loading}
                  onClick={() => changePage(page + 1)}
                  size="sm"
                  variant="outline"
                >
                  Next →
                </Button>
              </nav>
            </>
          )}
        </section>
      )}
    </div>
  )
}
