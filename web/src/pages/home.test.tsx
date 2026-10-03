import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react"
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

  fireEvent.change(screen.getByLabelText("Language"), { target: { value: "fr" } })

  await waitFor(() => expect(lastCall()).toMatchObject({ language: "fr", page: 1 }))
  expect(window.location.search).toBe("?topic=psychology&language=fr")
})

it("pages forward and back while keeping the applied state", async () => {
  renderHome()
  await submitTopic("psychology")
  fireEvent.change(screen.getByLabelText("Language"), { target: { value: "fr" } })
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
