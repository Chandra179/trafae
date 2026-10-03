import type { Book } from "@/api/books"

export const GENRES = [
  "non-fiction",
  "history",
  "science",
  "psychology",
  "philosophy",
  "biography",
  "economics",
  "business",
  "politics",
  "sociology",
  "education",
  "travel",
  "health",
  "technology",
  "religion",
] as const

export const GENRE_LABELS: Record<string, string> = {
  "non-fiction": "Non-fiction (all)",
}

export function genreLabel(genre: string) {
  return GENRE_LABELS[genre] ?? (genre.charAt(0).toUpperCase() + genre.slice(1))
}

const PROVIDER_LABELS: Record<string, string> = {
  open_library: "Open Library",
  doab: "DOAB",
  gutendex: "Gutendex",
  internet_archive: "Internet Archive",
  library_of_congress: "Library of Congress",
  wikidata: "Wikidata",
  project_gutenberg: "Project Gutenberg",
}

export function providerLabel(id: string) {
  return PROVIDER_LABELS[id] ?? id
}

const compact = new Intl.NumberFormat("en", { notation: "compact" })

export function formatCount(value: number) {
  return compact.format(value)
}

export function formatAuthors(book: Book) {
  if (!book.authors || book.authors.length === 0) return "Unknown author"
  const first = book.authors[0]
  if (book.authors.length === 1) return first
  return `${first} +${book.authors.length - 1}`
}

export function readNowUrl(book: Book) {
  return book.url ?? book.access_urls?.[0] ?? ""
}

export function licenseLabel(license: string) {
  const value = license.toLowerCase()
  if (value.includes("public-domain") || value.includes("public domain")) return "Public domain"
  if (value.includes("creativecommons")) {
    const match = /licenses\/([\w-]+)/.exec(value)
    return match ? `CC ${match[1].toUpperCase()}` : "Creative Commons"
  }
  return license.length > 24 ? `${license.slice(0, 24)}…` : license
}

