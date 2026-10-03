import { apiRequest } from "./client"

export type BookMetric = {
  value: number
  metric: string
}

export type BookRating = {
  value: number
  scale: number
  count?: number
}

export type BookSource = {
  provider: string
  id?: string
  url?: string
  popularity?: BookMetric
  rating?: BookRating
}

export type Book = {
  id?: string
  title: string
  authors?: string[]
  description?: string
  year?: number
  year_kind?: string
  genres?: string[]
  subjects?: string[]
  languages?: string[]
  isbns?: string[]
  url?: string
  cover_url?: string
  license?: string
  access_urls?: string[]
  popularity?: BookMetric
  rating?: BookRating
  source: BookSource
}

export type SearchResult = {
  book: Book
  sources: BookSource[]
  rrf_score: number
}

export type ProviderStatus = {
  provider: string
  status: "ok" | "partial" | "skipped" | "error"
  reason?: string
  count?: number
}

export type SearchResponse = {
  results: SearchResult[]
  providers: ProviderStatus[]
  limit: number
  page?: number
  has_more?: boolean
}

export type BookSearchParams = {
  topics: string[]
  genre?: string
  providers?: string[]
  minYear?: number
  maxYear?: number
  language?: string
  minRating?: number
  limit?: number
  page?: number
}

export function searchBooks(params: BookSearchParams, options?: { signal?: AbortSignal }) {
  const query = new URLSearchParams()
  for (const topic of params.topics) {
    query.append("topic", topic)
  }
  for (const provider of params.providers ?? []) {
    query.append("provider", provider)
  }
  if (params.genre) query.set("genre", params.genre)
  if (params.minYear !== undefined) query.set("min_year", String(params.minYear))
  if (params.maxYear !== undefined) query.set("max_year", String(params.maxYear))
  if (params.language) query.set("language", params.language)
  if (params.minRating !== undefined) query.set("min_rating", String(params.minRating))
  if (params.limit !== undefined) query.set("limit", String(params.limit))
  if (params.page !== undefined && params.page > 1) query.set("page", String(params.page))
  return apiRequest<SearchResponse>(`/books/search?${query.toString()}`, {
    signal: options?.signal,
  })
}
