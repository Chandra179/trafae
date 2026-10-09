---
title: "Trafae"
description: "Trafae searches several open book catalogs and combines their results into one ranked list, with direct book links where available."
seoTitle: "Trafae: Search Multiple Open Book Collections"
seoDescription: "Trafae searches multiple book catalogs, removes duplicates, and provides direct book links where available."
answerSummary: "Trafae queries seven open book catalogs at once, removes duplicate books, and merges the rankings with Reciprocal Rank Fusion. Cards link directly to books where a catalog provides a link, and to the catalog record otherwise."
tags: [books, search, open-access, aggregation]
links:
  github: "https://github.com/Chandra179/trafae"
created: 2026-10-03
updated: 2026-10-09
---

# Trafae: Search Multiple Open Book Collections

Search service that queries seven open book catalogs at once and returns one ranked, deduplicated list. No accounts, and it hosts no books.

## Algorithms and approach

```text
Search or genre browse -> Scatter to 7 catalogs (per-source timeout) -> Gather
  -> Deduplicate (identifiers, title, author) -> RRF fusion -> Filter -> Page
  -> Card link: direct book link, else catalog link
```

1. **Scatter-gather:** All catalogs are queried in parallel, each with its own time limit. A slow or blocked catalog only removes its own results.
2. **Deduplicate:** The same book from several catalogs is matched by identifiers, title and author, and merged into one entry that keeps each source's downloads, ratings and links.
3. **Rank:** **Reciprocal Rank Fusion (RRF)** combines the catalogs' orderings. It uses positions, not scores, so catalogs with incompatible scoring (downloads, stars, plain order) can be merged.
4. **Filter:** Language (aliases like `en`, `eng`, `english` are equal), year range, popularity floor and rating floor. Subjects match whole words, so "history" does not match "prehistory".
5. **Page:** A larger result pool is fetched and sorted once, then reused so page order stays stable. Catalogs run out at about 10 pages of 24 books.
6. **Link:** **Read / download** for a direct book link, **View source** for a catalog record, **No link available** otherwise.

## Sources

| Source | Offers |
|---|---|
| Open Library | Lending catalog with community ratings |
| Project Gutenberg | Public-domain books |
| Gutendex | Search API for Project Gutenberg |
| Internet Archive | Digital library with download counts and ratings |
| DOAB | Scholarly open-access books |
| Library of Congress | US national library digital collections |
| Wikidata | Community-maintained book records |

## Privacy

No accounts and no personal data. Searches, cache hits and clicks on direct book links are counted anonymously in memory and reset on restart. Searches are cached for a few minutes.

## Known limitations

- DOAB and Library of Congress block some hosting-provider networks.
- Gutendex is occasionally slow.
- Open Library sometimes serves blank cover images.
- Very deep result pages may be empty.

## References

- [Reciprocal Rank Fusion paper](https://doi.org/10.1145/1571941.1572114).
- [Scatter-Gather pattern](https://www.enterpriseintegrationpatterns.com/patterns/messaging/BroadcastAggregate.html).
