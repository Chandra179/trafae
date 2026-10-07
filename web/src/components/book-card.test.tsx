import { cleanup, fireEvent, render, screen } from "@testing-library/react"
import { afterEach, beforeEach, expect, it, vi } from "vitest"

import type { Book, SearchResult } from "@/api/books"
import { sendAccessClick } from "@/api/events"
import { BookCard } from "@/components/book-card"

vi.mock("@/api/events", () => ({ sendAccessClick: vi.fn() }))

afterEach(cleanup)
beforeEach(() => vi.mocked(sendAccessClick).mockClear())

function result(overrides: Partial<Book> = {}): SearchResult {
  return {
    book: {
      title: "Biology",
      url: "https://catalog.test/book",
      source: { provider: "open_library" },
      ...overrides,
    },
    sources: [{ provider: "open_library" }, { provider: "gutendex" }],
    rrf_score: 0.03,
  }
}

it("opens direct content in a new tab and records the actual provider", () => {
  render(<BookCard result={result({ access_urls: ["https://books.test/book.html"] })} />)
  const link = screen.getByRole("link", { name: "Read / download" })
  expect(link.getAttribute("href")).toBe("https://books.test/book.html")
  expect(link.getAttribute("target")).toBe("_blank")
  expect(link.getAttribute("rel")).toBe("noreferrer")
  fireEvent.click(link)
  expect(sendAccessClick).toHaveBeenCalledExactlyOnceWith("gutendex")
})

it("opens a catalog page without recording a reading click", () => {
  render(<BookCard result={result()} />)
  const link = screen.getByRole("link", { name: "View source" })
  expect(link.getAttribute("href")).toBe("https://catalog.test/book")
  expect(link.getAttribute("target")).toBe("_blank")
  fireEvent.click(link)
  expect(sendAccessClick).not.toHaveBeenCalled()
})

it("offers the first distinct alternate format and records its click", () => {
  render(<BookCard result={result({
    access_urls: [
      "https://books.test/book.html", "https://books.test/book.html",
      "javascript:alert(1)", "https://books.test/book.epub",
    ],
  })} />)
  const alternate = screen.getByRole("link", { name: "Another format" })
  expect(alternate.getAttribute("href")).toBe("https://books.test/book.epub")
  expect(alternate.getAttribute("target")).toBe("_blank")
  fireEvent.click(alternate)
  expect(sendAccessClick).toHaveBeenCalledExactlyOnceWith("gutendex")
})

it("hides the alternate action when all direct links are duplicates", () => {
  render(<BookCard result={result({
    access_urls: ["https://books.test/book.html", "https://books.test/book.html"],
  })} />)
  expect(screen.queryByRole("link", { name: "Another format" })).toBeNull()
})

it("shows an unavailable state when there is no usable link", () => {
  render(<BookCard result={result({ url: "javascript:alert(1)", access_urls: ["/relative"] })} />)
  expect(screen.getByText("No link available")).toBeTruthy()
  expect(screen.queryByRole("link")).toBeNull()
})
