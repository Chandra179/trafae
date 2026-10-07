import { expect, it } from "vitest"

import type { Book, SearchResult } from "@/api/books"
import { bookAccessAction } from "@/lib/book-display"

function result(overrides: Partial<Book> = {}): SearchResult {
  return {
    book: {
      title: "Biology",
      url: "https://catalog.test/book",
      source: { provider: "open_library", url: "https://catalog.test/source" },
      ...overrides,
    },
    sources: [{ provider: "open_library" }, { provider: "gutendex" }],
    rrf_score: 0.03,
  }
}

it("prefers direct content and attributes merged links to Gutendex", () => {
  expect(bookAccessAction(result({ access_urls: ["https://books.test/book.html"] }))).toEqual({
    url: "https://books.test/book.html",
    label: "Read / download",
    alternateUrl: undefined,
    provider: "gutendex",
  })
})

it("skips invalid entries and selects a distinct alternate format", () => {
  expect(bookAccessAction(result({
    access_urls: [
      "", "javascript:alert(1)", "/relative", "https://",
      " https://books.test/book.html ", "https://books.test/book.html",
      "http://books.test/book.epub",
    ],
  }))).toMatchObject({
    url: "https://books.test/book.html",
    alternateUrl: "http://books.test/book.epub",
  })
})

it("labels catalog-only links as source pages without a reading event provider", () => {
  expect(bookAccessAction(result())).toEqual({
    url: "https://catalog.test/book",
    label: "View source",
  })
})

it.each(["", "javascript:alert(1)", "/relative", "https://", "https:catalog.test/book"])(
  "falls back to source.url when book.url is unusable (%s)",
  (url) => {
    expect(bookAccessAction(result({ url, access_urls: ["file:///book"] }))).toEqual({
      url: "https://catalog.test/source",
      label: "View source",
    })
  },
)

it("returns no action when all links are missing or invalid", () => {
  expect(bookAccessAction(result({
    url: "",
    access_urls: ["//books.test/book", "data:text/html,book"],
    source: { provider: "wikidata", url: "javascript:alert(1)" },
  }))).toBeUndefined()
})

it("uses the direct record provider when there is no merged Gutendex source", () => {
  const direct = result({ access_urls: ["https://books.test/book.pdf"] })
  direct.sources = [{ provider: "open_library" }]
  expect(bookAccessAction(direct)?.provider).toBe("open_library")
})
