import { apiBaseUrl } from "./client"

// sendAccessClick reports that a reader opened one of a book's read links.
// It is a fire-and-forget beacon: it must never block or break the
// navigation, so all failures are silently ignored.
export function sendAccessClick(provider: string | undefined): void {
  if (typeof navigator === "undefined" || typeof navigator.sendBeacon !== "function") {
    return
  }
  const payload = JSON.stringify({ type: "access_click", provider: provider ?? "" })
  try {
    navigator.sendBeacon(`${apiBaseUrl}/books/events`, new Blob([payload], { type: "application/json" }))
  } catch {
    // The funnel metric must never cost the user their click.
  }
}
