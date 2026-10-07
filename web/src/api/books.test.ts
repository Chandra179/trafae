import { afterEach, expect, it, vi } from "vitest"

import { searchBooks } from "@/api/books"
import { ApiError } from "@/api/client"

afterEach(() => vi.unstubAllGlobals())

it("sends the applied search, filters, and pagination through the API client", async () => {
  const response = { results: [], providers: [], limit: 24, page: 2, has_more: false }
  const fetch = vi.fn().mockResolvedValue(new Response(JSON.stringify(response)))
  vi.stubGlobal("fetch", fetch)
  const controller = new AbortController()
  const result = await searchBooks({
    topics: ["biology", "ecology"], genre: "science", providers: ["gutendex"],
    language: "en", minYear: 1900, maxYear: 1999, minRating: 4, limit: 24, page: 2,
  }, { signal: controller.signal })
  const [requestUrl, options] = fetch.mock.calls[0]
  const query = new URL(requestUrl, "https://trafae.test").searchParams
  expect(query.getAll("topic")).toEqual(["biology", "ecology"])
  expect(Object.fromEntries(query)).toMatchObject({
    genre: "science", provider: "gutendex", language: "en",
    min_year: "1900", max_year: "1999", min_rating: "4", limit: "24", page: "2",
  })
  expect(options.signal).toBe(controller.signal)
  expect(result).toEqual(response)
})

it("surfaces the API's total provider failure instead of treating it as empty results", async () => {
  vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(
    JSON.stringify({ error: "all eligible book providers failed" }), { status: 503 },
  )))
  await expect(searchBooks({ topics: ["biology"] })).rejects.toMatchObject({
    name: "ApiError", status: 503, message: "all eligible book providers failed",
  } satisfies Partial<ApiError>)
})
