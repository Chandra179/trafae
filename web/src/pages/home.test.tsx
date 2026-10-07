import { act, cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react"
import { BrowserRouter } from "react-router-dom"
import { afterEach, beforeEach, expect, it, vi } from "vitest"

import { searchBooks, type SearchResponse } from "@/api/books"
import { HomePage } from "@/pages/home"

vi.mock("@/api/books", () => ({ searchBooks: vi.fn() }))

const searchBooksMock = vi.mocked(searchBooks)

// vitest does not run testing-library's auto-cleanup without globals: true.
afterEach(cleanup)

function okResponse(overrides: Partial<SearchResponse> = {}): SearchResponse {
  return {
    results: [
      {
        book: {
          title: "Psychology of Everything",
          authors: ["Ada Author"],
          subjects: ["Psychology"],
          source: { provider: "gutendex", id: "42", url: "https://gutenberg.test/42" },
        },
        sources: [{ provider: "gutendex", id: "42" }],
        rrf_score: 0.03,
      },
    ],
    providers: [{ provider: "gutendex", status: "ok", count: 1 }],
    limit: 24,
    has_more: true,
    ...overrides,
  }
}

function renderHome() {
  return render(
    <BrowserRouter>
      <HomePage />
    </BrowserRouter>,
  )
}

async function submitTopic(topic: string) {
  fireEvent.change(screen.getByPlaceholderText(/search a topic/i), { target: { value: topic } })
  fireEvent.click(screen.getByRole("button", { name: /search the stacks/i }))
  await waitFor(() => expect(searchBooksMock).toHaveBeenCalled())
}

function lastCall() {
  return searchBooksMock.mock.calls[searchBooksMock.mock.calls.length - 1][0]
}

beforeEach(() => {
  window.history.replaceState(null, "", "/")
  searchBooksMock.mockReset()
  searchBooksMock.mockResolvedValue(okResponse())
})

it("commits the topic on submit without writing the default genre to the URL", async () => {
  renderHome()
  await submitTopic("psychology")

  expect(window.location.search).toBe("?topic=psychology")
  expect(lastCall()).toMatchObject({ topics: ["psychology"], genre: "non-fiction", page: 1 })
})

it("browses a genre from a chip click without sweeping in uncommitted text", async () => {
  renderHome()
  fireEvent.change(screen.getByPlaceholderText(/search a topic/i), { target: { value: "typed but not submitted" } })
  fireEvent.click(screen.getByRole("button", { name: "History" }))

  await waitFor(() => expect(searchBooksMock).toHaveBeenCalled())
  expect(window.location.search).toBe("?genre=history")
  expect(lastCall()).toMatchObject({ topics: [], genre: "history" })
})

it("applies filter edits immediately and returns to page 1", async () => {
  renderHome()
  await submitTopic("psychology")
  await waitFor(() => screen.getByRole("button", { name: /next/i }))
  fireEvent.click(screen.getByRole("button", { name: /next/i }))
  await waitFor(() => expect(lastCall()).toMatchObject({ page: 2 }))

  fireEvent.click(screen.getByRole("button", { name: /^Language/ }))
  fireEvent.click(screen.getByRole("option", { name: "French" }))

  await waitFor(() => expect(lastCall()).toMatchObject({ language: "fr", page: 1 }))
  expect(window.location.search).toBe("?topic=psychology&language=fr")
})

it("applies a year preset through the dropdown and shows it as a removable chip", async () => {
  renderHome()
  await submitTopic("psychology")
  await waitFor(() => screen.getByRole("button", { name: /^Published/ }))

  fireEvent.click(screen.getByRole("button", { name: /^Published/ }))
  fireEvent.click(screen.getByRole("option", { name: "21st century" }))

  await waitFor(() => expect(lastCall()).toMatchObject({ minYear: 2000, page: 1 }))
  expect(window.location.search).toBe("?topic=psychology&min_year=2000")
  expect(screen.getByRole("button", { name: "Remove Published filter" })).toBeTruthy()

  fireEvent.click(screen.getByRole("button", { name: "Remove Published filter" }))

  await waitFor(() => expect(lastCall()).toMatchObject({ minYear: undefined, maxYear: undefined }))
  expect(window.location.search).toBe("?topic=psychology")
})

it("pages forward and back while keeping the applied state", async () => {
  renderHome()
  await submitTopic("psychology")
  await waitFor(() => screen.getByRole("button", { name: /^Language/ }))
  fireEvent.click(screen.getByRole("button", { name: /^Language/ }))
  fireEvent.click(screen.getByRole("option", { name: "French" }))
  await waitFor(() => expect(lastCall()).toMatchObject({ language: "fr" }))

  const next = screen.getByRole("button", { name: /next/i })
  fireEvent.click(next)
  await waitFor(() => expect(lastCall()).toMatchObject({ page: 2, language: "fr", topics: ["psychology"] }))
  expect(window.location.search).toBe("?topic=psychology&language=fr&page=2")

  fireEvent.click(screen.getByRole("button", { name: /previous/i }))
  await waitFor(() => expect(lastCall()).toMatchObject({ page: 1 }))
  expect(window.location.search).toBe("?topic=psychology&language=fr")
})

it("disables Previous on the first page", async () => {
  renderHome()
  await submitTopic("psychology")
  await waitFor(() => screen.getByRole("button", { name: /previous/i }))

  expect(screen.getByRole("button", { name: /previous/i }).hasAttribute("disabled")).toBe(true)
  expect(screen.getByRole("button", { name: /next/i }).hasAttribute("disabled")).toBe(false)
})

it("re-syncs the topic input when navigating back", async () => {
  renderHome()
  await submitTopic("psychology")
  await submitTopic("biology")
  expect((screen.getByPlaceholderText(/search a topic/i) as HTMLInputElement).value).toBe("biology")

  window.history.back()

  await waitFor(() =>
    expect((screen.getByPlaceholderText(/search a topic/i) as HTMLInputElement).value).toBe("psychology"),
  )
  expect(window.location.search).toBe("?topic=psychology")
  // The URL change is a new search: the page re-fetches.
  await waitFor(() => expect(lastCall()).toMatchObject({ topics: ["psychology"] }))
})

it("applies and removes a rating filter while preserving language", async () => {
  renderHome()
  await submitTopic("psychology")
  await waitFor(() => screen.getByRole("button", { name: /^Language/ }))
  fireEvent.click(screen.getByRole("button", { name: /^Language/ }))
  fireEvent.click(screen.getByRole("option", { name: "French" }))
  await waitFor(() => expect(lastCall()).toMatchObject({ language: "fr" }))
  fireEvent.click(screen.getByRole("button", { name: /^Minimum rating/ }))
  fireEvent.click(screen.getByRole("option", { name: "4.0 & up" }))
  await waitFor(() => expect(lastCall()).toMatchObject({ minRating: 4, language: "fr", page: 1 }))
  expect(window.location.search).toBe("?topic=psychology&language=fr&min_rating=4")

  fireEvent.click(screen.getByRole("button", { name: "Remove Rating filter" }))
  await waitFor(() => expect(lastCall()).toMatchObject({ minRating: undefined, language: "fr" }))
})

it("disables Next on the final page", async () => {
  searchBooksMock.mockResolvedValue(okResponse({ has_more: false }))
  renderHome()
  await submitTopic("psychology")
  await waitFor(() => screen.getByRole("button", { name: /next/i }))
  expect(screen.getByRole("button", { name: /next/i }).hasAttribute("disabled")).toBe(true)
})

it("shows empty results and lets a subsequent search recover", async () => {
  searchBooksMock.mockResolvedValueOnce(okResponse({ results: [], has_more: false }))
  renderHome()
  await submitTopic("missing")
  await screen.findByText("No books matched")
  expect(screen.queryByRole("navigation", { name: "Result pages" })).toBeNull()
  await submitTopic("psychology")
  await screen.findByText("Psychology of Everything")
})

it("lets readers return from a page beyond the end", async () => {
  window.history.replaceState(null, "", "/?topic=psychology&page=3")
  searchBooksMock.mockResolvedValueOnce(okResponse({ results: [], has_more: false }))
  renderHome()
  await screen.findByText("This page is past the end of the results")
  expect(screen.getByRole("button", { name: /next/i }).hasAttribute("disabled")).toBe(true)
  fireEvent.click(screen.getByRole("button", { name: /previous/i }))
  await waitFor(() => expect(lastCall()).toMatchObject({ page: 2 }))
  await screen.findByText("Psychology of Everything")
})

it("keeps partial results and explains failed and skipped catalogs", async () => {
  searchBooksMock.mockResolvedValue(okResponse({
    providers: [
      { provider: "gutendex", status: "partial", count: 1, reason: "One topic failed" },
      { provider: "doab", status: "error", count: 0, reason: "HTTP 403" },
      { provider: "wikidata", status: "skipped", count: 0, reason: "rating not supported" },
    ],
  }))
  renderHome()
  await submitTopic("psychology")
  await screen.findByText("Psychology of Everything")
  expect(screen.getByText(/Results from Gutendex \(partial\)/)).toBeTruthy()
  expect(screen.getByText("1 catalog unavailable")).toBeTruthy()
  expect(screen.getByText("DOAB couldn't return results.")).toBeTruthy()
  expect(screen.getByText("DOAB: HTTP 403")).toBeTruthy()
  expect(screen.getByText("1 catalog skipped")).toBeTruthy()
  expect(screen.getByText("Wikidata: rating not supported")).toBeTruthy()
  expect(screen.queryByText(/didn't answer in time/)).toBeNull()
})

it("shows a total search failure and recovers with another query", async () => {
  searchBooksMock.mockRejectedValueOnce(new Error("all eligible book providers failed"))
  renderHome()
  await submitTopic("unavailable")
  await screen.findByText("Search failed")
  expect(screen.getByText("all eligible book providers failed")).toBeTruthy()
  await submitTopic("psychology")
  await screen.findByText("Psychology of Everything")
  expect(screen.queryByText("Search failed")).toBeNull()
})

it("ignores an old response after a newer search has finished", async () => {
  let finishOld!: (response: SearchResponse) => void
  searchBooksMock.mockImplementationOnce(() => new Promise((resolve) => { finishOld = resolve }))
  renderHome()
  await submitTopic("old topic")
  await submitTopic("psychology")
  await screen.findByText("Psychology of Everything")
  const oldSignal = searchBooksMock.mock.calls[0][1]?.signal
  expect(oldSignal?.aborted).toBe(true)

  await act(async () => {
    finishOld(okResponse({ results: [], has_more: false }))
  })
  expect(screen.getByText("Psychology of Everything")).toBeTruthy()
  expect(screen.queryByText("Searching the stacks…")).toBeNull()
})
