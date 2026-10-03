import { useState } from "react"

import type { Book, SearchResult } from "@/api/books"
import { Card, CardContent, CardFooter, CardHeader } from "@/components/ui/card"
import {
  formatAuthors,
  formatCount,
  licenseLabel,
  providerLabel,
  readNowUrl,
} from "@/lib/book-display"

// Book-spine gradients matched to the Editorial Library prototype (c1–c5).
const COVER_PALETTES = [
  "linear-gradient(160deg, #3d6b4f, #1f3c2b)",
  "linear-gradient(160deg, #7b4a2f, #4a2a18)",
  "linear-gradient(160deg, #39506e, #1d2c42)",
  "linear-gradient(160deg, #6e3950, #3c1e2c)",
  "linear-gradient(160deg, #8a6d2f, #54401a)",
]

function coverPalette(title: string) {
  let hash = 0
  for (let i = 0; i < title.length; i++) hash = (hash * 31 + title.charCodeAt(i)) | 0
  return COVER_PALETTES[Math.abs(hash) % COVER_PALETTES.length]
}

function initials(title: string) {
  return title
    .split(/\s+/)
    .filter((word) => /[a-z0-9]/i.test(word))
    .slice(0, 2)
    .map((word) => word.charAt(0).toUpperCase())
    .join("")
}

function BookCover({ book }: { book: Book }) {
  const [failed, setFailed] = useState(false)
  if (book.cover_url && !failed) {
    return (
      <img
        alt={`Cover of ${book.title}`}
        className="h-full w-full object-cover"
        decoding="async"
        loading="lazy"
        onError={() => setFailed(true)}
        src={book.cover_url}
      />
    )
  }
  return (
    <div
      className="flex h-full w-full items-center justify-center font-serif text-[26px] font-bold text-white/90 shadow-[inset_-6px_0_10px_rgba(0,0,0,0.14)]"
      style={{ background: coverPalette(book.title) }}
    >
      {initials(book.title)}
    </div>
  )
}

export function BookCard({ result }: { result: SearchResult }) {
  const { book, sources } = result
  const readUrl = readNowUrl(book)
  const catalogCount = sources.length

  return (
    <Card className="h-full gap-0 overflow-hidden rounded-lg border-border/80 py-0 shadow-[0_1px_2px_rgba(60,50,30,0.05)]">
      <div className="flex gap-3.5 border-b border-dashed border-border p-4">
        <div className="h-[122px] w-[86px] shrink-0 overflow-hidden rounded-[3px] shadow-[0_2px_5px_rgba(50,40,20,0.18)]">
          <BookCover book={book} />
        </div>
        <CardHeader className="min-w-0 flex-1 gap-0 p-0">
          <h3 className="break-words font-serif text-[17px] font-bold leading-[1.3]">{book.title}</h3>
          <p className="mt-0.5 font-serif text-sm italic text-muted-foreground">
            {formatAuthors(book)}
          </p>
          <div className="mt-2 flex flex-wrap items-center gap-x-2.5 gap-y-1 font-sans text-xs text-muted-foreground">
            {book.year !== undefined && <span>{book.year}</span>}
            {book.rating && (
              <span>
                ★ {book.rating.value.toFixed(1)}
                {book.rating.count ? ` (${formatCount(book.rating.count)})` : ""}
              </span>
            )}
            {book.popularity && (
              <span>
                {formatCount(book.popularity.value)} {book.popularity.metric}
              </span>
            )}
            {book.license && (
              <span className="rounded-[3px] border border-border bg-secondary px-[7px] py-px font-semibold text-primary">
                {licenseLabel(book.license)}
              </span>
            )}
          </div>
        </CardHeader>
      </div>

      <CardContent className="flex flex-1 flex-col gap-2 px-4 pt-3">
        {book.description && (
          <p className="line-clamp-4 text-sm leading-snug text-muted-foreground">{book.description}</p>
        )}
        {book.subjects && book.subjects.length > 0 && (
          <p className="truncate font-sans text-xs text-muted-foreground/80">
            Subjects: {book.subjects.slice(0, 4).join(", ")}
          </p>
        )}
        <p className="mt-auto font-sans text-xs text-muted-foreground">
          Matched by{" "}
          <span className="font-semibold text-foreground">
            {catalogCount} {catalogCount === 1 ? "catalog" : "catalogs"}
          </span>{" "}
          — {sources.map((source) => providerLabel(source.provider)).join(", ")}
        </p>
      </CardContent>

      <CardFooter className="gap-2 px-4 pb-4 pt-3">
        {readUrl ? (
          <a
            className="inline-flex items-center gap-2 rounded-md bg-foreground px-4 py-2 font-sans text-sm font-semibold text-background transition-colors hover:bg-foreground/90"
            href={readUrl}
            rel="noreferrer"
            target="_blank"
          >
            Read now
          </a>
        ) : (
          <span className="inline-flex items-center rounded-md bg-secondary px-4 py-2 font-sans text-sm font-medium text-muted-foreground">
            No direct link
          </span>
        )}
        {book.access_urls && book.access_urls.length > 1 && (
          <a
            className="font-sans text-[13px] font-medium text-primary underline underline-offset-[3px] hover:text-primary/80"
            href={book.access_urls[1]}
            rel="noreferrer"
            target="_blank"
          >
            More formats
          </a>
        )}
      </CardFooter>
    </Card>
  )
}
