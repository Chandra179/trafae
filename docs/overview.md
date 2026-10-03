---
title: "Trafae"
description: "Trafae is a book discovery service that searches several open book catalogs at once and merges them into one ranked list of free, legal reads."
seoTitle: "Trafae: One Search Across the Open Book Collections"
seoDescription: "Trafae searches multiple free book catalogs at once, removes duplicates, and fuses the results into a single ranked list."
answerSummary: "Trafae is a book discovery service that searches several open book catalogs at once and merges them into one ranked list of free, legal reads."
tags: [books, search, open-access, aggregation]
links:
  github: "https://github.com/Chandra179/trafae"
created: 2026-10-03
---

# Trafae: One Search Across the Open Book Collections

Trafae is a book discovery service for readers who want to go deep on a
topic and start reading immediately. Ask it for a subject or browse by
genre, and it searches several open book collections at the same time,
removes the duplicates, and hands back one merged, best-first list — where
every book can be read legally for free.

It is useful for:

- students and independent learners going deep on a subject;
- readers of public-domain classics and open-access scholarship;
- researchers looking for freely readable books on a topic; and
- anyone tired of opening three or four catalog websites per question.

Trafae needs no account. It does not host the books themselves — it points
to the collections that do, and every link it shows leads to a copy you can
read at no cost.

## How it works

```text
your search or genre browse
        │
        ▼
seven open book collections are asked at the same time
        │
        ▼
duplicates collapsed · rankings combined · best books first
        │
        ▼
browse the list, filter it, turn the pages
        │
        ▼
open the book where it can be read for free
```

Trafae presents this through a simple web page: a search box, a row of
genre shortcuts, and a list of book cards with covers, descriptions, and
each collection's popularity and rating evidence.

### 1. Search or browse

Type a topic — "psychology", "cartography", "ancient Rome" — or browse a
genre without typing anything at all. Either can be combined with filters:
language, publication year range, a popularity floor, or a minimum rating.

### 2. Every collection at once

The question goes out to all connected collections simultaneously, each
with its own time budget. This is a pattern called
[scatter-gather](https://www.enterpriseintegrationpatterns.com/patterns/messaging/BroadcastAggregate.html):
ask everyone at once, then gather whatever came back. A slow or unreachable
collection never holds the others hostage, and the page shows honestly
which collections answered and which did not.

### 3. One merged list

The same book found in several collections appears once, keeping every
source's evidence — downloads, ratings, links. The rankings from the
separate collections are then combined — a technique called
[Reciprocal Rank Fusion](https://doi.org/10.1145/1571941.1572114) — so
that books several collections agree on rise to the top.

### 4. Open and read

Each result links to where the book can be read or downloaded. When a
reader opens a book, an anonymous click is counted — the service's way of
learning whether its recommendations are actually being read.

## Main features

- **One search, many collections** — seven open book catalogs are searched
  in parallel with a single question.
- **Free and legal only** — every source offers books that can be read
  immediately at no cost: public domain or open access.
- **Merged, deduplicated results** — the same book from several catalogs
  collapses into one entry that keeps each source's ratings and popularity.
- **Fair ranking** — each catalog's own best-first ordering is combined
  into one list, so agreement across catalogs boosts a book without any
  single catalog dominating.
- **Genre browsing** — browse history, science, psychology and more with no
  search term at all; non-fiction is the default lens.
- **Filters that mean what they say** — language, year range, popularity
  floor, and rating floor are applied to the results you actually see.
- **Transparent sources** — every card shows where a book came from, and
  every search reports which collections answered.
- **Fast repeats and page turns** — a recent search is remembered for a few
  minutes, so revisiting it or flipping pages comes back instantly.
- **Stable paging** — ordering does not shuffle between pages, and when the
  collections have genuinely run out of books, the list says so instead of
  trailing off.
- **Shareable searches** — the address captures the search and its filters,
  so the back button works and a link shows a friend exactly what you saw.
- **Degrades gracefully** — a blocked or slow collection costs you its own
  results, not the whole search.

## Where the books come from

| Source | What it offers |
|---|---|
| Open Library | A vast lending catalog of published books, with community ratings |
| Project Gutenberg | Public-domain classics, straight from the official catalog, refreshed daily |
| Gutendex | A second, live window onto Project Gutenberg's own catalog |
| Internet Archive | A general digital library, with download counts and ratings |
| DOAB | Scholarly open-access books from academic publishers worldwide |
| Library of Congress | Digital collections from the US national library |
| Wikidata | Community-maintained book records covering subjects the others miss |

No single collection is trusted blindly: each contributes its own ordering,
and popularity or ratings are shown per source rather than silently blended.

## How the ranking works

### Combining the collections' opinions

Each collection returns its own best-first list, but they score books in
incompatible ways — one counts downloads, another uses star ratings, a
third just orders results. Trafae therefore compares *positions*, not
scores: a book that appears near the top of any collection's list earns
credit, and appearing high in several lists earns more. This technique,
known as Reciprocal Rank Fusion, is why the merged list feels reasonable
even though the sources measure nothing alike.

### Collapsing duplicates

The same book often appears in three or four collections under slightly
different records. Trafae recognizes it by its identifiers and by title and
author, keeps one entry, and preserves every source's evidence alongside
it — so "three catalogs agree" stays visible instead of being averaged
away.

### Evidence you can filter on

Download counts and star ratings are shown on the results and can be used
as floors: "only books rated four stars or better", "only books many people
have read". Publication year and language work the same way, and language
spellings are understood generously — "english", "eng", and "en" all mean
the same thing.

### Paging that stays fast

When a reader asks for page three, Trafae quietly collects a deeper pool
than it shows — a few extra pages' worth — sorts it once, hands out the
requested slice, and keeps the rest ready. The next page is then an instant
slice of an answer already in hand, with the ordering exactly as it was.
Only when a reader pages past that pool does Trafae fetch again, deeper.
Collections can only be asked so deep before they run out of answers
(roughly ten pages of twenty-four books); at that point the list ends
honestly rather than padding itself.

### Careful matching

Subjects are matched by whole words, so a search for "history" is not
flooded by "prehistory" and "cartography", and a browse for non-fiction
actively excludes anything marked fiction. These small rules keep genre
browsing from drifting.

## Privacy

Trafae has no accounts and stores no personal data. It counts two kinds of
anonymous things: how many searches were made (and how many were answered
from memory), and how many times a reader opened a book's read link. That
last number — the share of searches that end in a book being opened — is
the service's own measure of usefulness. All counters live in memory and
reset when the service restarts.

## Known limitations

- **Some catalogs refuse some networks.** DOAB and the Library of Congress
  block certain hosting-provider addresses; from such a network those
  sources simply go quiet. From a home or office connection they usually
  work.
- **Gutendex is occasionally slow.** It has its own time budget and
  recovers on the next search.
- **Covers can be blank.** Open Library sometimes serves an empty cover
  image that still loads successfully, so a book card may show a blank
  where a cover should be.
- **Deep pages run dry.** Because collections cap how deep they can be
  asked, very deep result pages end earlier than an eager reader might
  like — the trade-off for keeping ordering stable and honest.

## References

- Gordon V. Cormack, Charles L. A. Clarke, and Stefan Büttcher (2009).
  [Reciprocal Rank Fusion Outperforms Condorcet and Individual Rank Learning
  Methods](https://doi.org/10.1145/1571941.1572114). *SIGIR '09*,
  pages 759–760 — the paper behind the ranking described in "How the
  ranking works".
- Gregor Hohpe and Bobby Woolf (2003). [Scatter-Gather](https://www.enterpriseintegrationpatterns.com/patterns/messaging/BroadcastAggregate.html).
  *Enterprise Integration Patterns* — the pattern named in "How it
  works".
