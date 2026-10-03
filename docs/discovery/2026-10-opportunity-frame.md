# Lux — Opportunity Frame & Assessment

Date: 2026-10-02 · Status: hypothesis, zero customer evidence yet · Method: INSPIRED (Cagan) discovery

> **Owner decision, 2026-10-03: launch-and-learn.** The value-test interview
> round was skipped (its guide has been removed). The value hypothesis stays
> unvalidated; real usage is now the evidence channel. Launch success criteria
> and the search→read funnel are instrumented server-side instead — see the
> `/metrics` endpoint and the launch-readiness section of TODO.md. Pass/fail
> numbers from §3 remain the reference point, measured by funnel data rather
> than interviews.

---

## 1. Problem frame (Workflow 1)

### Customer (hypothesis — to be validated in interviews)

**Deliberate self-learners and serious readers searching for non-fiction** —
people who want to go deep on a topic and prefer (or need) books they can
legally read immediately for free: students, researchers-adjacent readers,
indie learners, public-domain enthusiasts.

This segment is inferred from the product's own built decisions, not from
customer contact:

| Built decision | What it implies about the intended customer |
|---|---|
| `default_genre: "non-fiction"` in config | Casual fiction browsers are not the target |
| DOAB (scholarly open access) is a first-class source | Academic/learning-oriented use |
| Project Gutenberg + Internet Archive sources | Public-domain / legally-free full text matters |
| `min_year`, `min_popularity`, `min_rating` filters | Quality- and recency-driven search, not browsing |
| RRF fusion + per-source metrics kept in results | "Multiple catalogs agree" as a trust signal |

### Problem (in their words — placeholder, to be replaced by interview quotes)

> "I end up searching Goodreads for what's *good*, then separately checking
> whether I can actually get the book free — Gutenberg, Archive.org, DOAB —
> three or four tabs per topic."

⚠️ This is a constructed illustration of the hypothesized pain, **not** a real
customer quote. First interviews replace it.

### Job to be done

**When** I want to go deep on a topic, **I want to** find the best books on it
that I can start reading right now for free, **so I can** spend my time
reading instead of hunting across sites and paywalls.

### Strategy fit

Lux's de-facto strategy (from the code) is: *aggregate only legally-free,
openly-licensed content; rank by cross-catalog agreement plus popularity and
rating evidence; stay transparent about where every book came from.* The
framed problem sits squarely inside that. No conflict.

### Explicit hypothesis

**We believe** deliberate self-learners searching for non-fiction have a
frequent, annoying problem: finding quality books they can actually access
means juggling Google, Goodreads, and each open catalog by hand.
**We believe** one fused search across open catalogs — deduplicated, ranked by
agreement, with per-source popularity/rating shown — will achieve readers
starting a worth-reading book in minutes instead of abandoning the hunt.
**We'll know it's true when** in a behavior test with real users (not opinions),
a majority pick the fused result list over their current method for real
topics AND describe their current method as multi-site, repeated, and
dissatisfying.

### Four risks — current state

| Risk | Status | Why |
|---|---|---|
| **Value** | 🔴 **Biggest unknown** | No evidence anyone wants fused catalog search vs. settling for Google/Goodreads |
| Usability | 🟡 Untested | No UI exists; the API is powerful but expert-shaped (RRF scores, min_popularity) |
| Feasibility | 🟢 Largely de-risked | Backend works end-to-end: 7 providers, fusion, dedup, daily catalog sync, tested |
| Business viability | 🟡 Unposed | No model in the repo; legal posture is strong (public domain / open access only); for an OSS project viability ≈ hosting cost + maintainer motivation |

---

## 2. Opportunity assessment (Workflow 4)

Script: `opportunity-assessment.mjs`. Scores are honest estimates given zero
customer evidence — interviews will move them.

| # | Question | Score | Rationale |
|---|---|---|---|
| 1 | Real, frequent, painful problem? | 3 | The multi-site hunt is real for heavy readers; risk that most people simply settle for Google/Goodreads |
| 2 | Hated workaround today? | 3 | Workarounds exist and are fragmented, but they're tolerable — pain may be minor, not hated |
| 3 | Strategy/principles fit? | 4 | The entire stack was built around open catalogs; problem and product are aligned |
| 4 | Feasibility + viability? | 3 | Feasibility strong (core built & tested); viability unposed (no model, OSS hosting costs) |
| 5 | Payoff worth discovery effort? | 4 | Discovery is cheap — backend already runs, so a live-data test needs only a thin UI; big-if-true in the open-knowledge niche |
| | **Total** | **17/25** | |

**Verdict: 15–25 → worth a discovery cycle. Proceed to problem interviews.**

---

## 3. Discovery cycle handoff (Workflow 3, value risk)

Script: `discovery-planner.mjs --risk value`. Cheapest test that answers the
value risk for THIS project is a **live-data prototype** (not just concierge):
the `/books/search` API already works, so a minimal one-page search UI on real
data is a few days of work — while a concierge test needs zero code.

**Test plan**

1. Recruit 5–8 deliberate self-learners from real segments (self-education
   communities, OA/academic-adjacent forums) — not friends/family.
2. For each participant: 2 real topics of their choosing. First watch them
   find books their usual way (behavior, frequency, pain). Do not pitch Lux.
   Then let them run the same topics through the prototype.
3. Pass/fail criteria, defined before testing:
   - **PASS:** ≥ 4 of 6 participants (a) describe a current method spanning
     2+ catalog sites per topic, and (b) start reading (or save) a book from
     the fused list they hadn't found their own way.
   - **FAIL:** < 3 of 6, or most say their current method "is fine" — the
     problem is convenience, not pain. Iterate on framing or kill.
4. Decide: **iterate, pivot, or kill** — killing a wrong idea is a success.

**Do NOT build before this test returns.** The frontend is currently a
template; that is the correct state until value evidence exists.

---

## 4. First product slice & idea backlog (owner direction: 2026-10-02)

Owner decision: the product is **book search and genre discovery** first.
This slice doubles as the live-data prototype for the value test above —
building it is the test.

### Slice 1 — search books, find by genre (build now)

Everything below is already served by `/books/search` + `/books/providers`;
this is frontend-only work:

1. **Topic search box** — `?topic=` (up to 8 terms).
2. **Genre browse** — curated genre chips/pages. The de-facto taxonomy already
   exists in `books/search.go` (`history, science, psychology, philosophy,
   biography, economics, business, politics, sociology, education, travel,
   health, technology, religion`) plus the `non-fiction` default.
3. **Filters** — year range (`min_year`/`max_year`), `language`,
   `min_rating`, `min_popularity`, `limit` (≤ 50), `provider` selection.
4. **Result cards** — title, authors, year, cover, license badge, and "read
   now" links from `url` / `access_urls`.
5. **Transparency strip** — show `providers` status (ok / skipped / error)
   and per-result `sources` + `rrf_score`: "why this book is here" is data
   the API already returns.

### Slice 2 — after a value signal (small backend work)

6. Sort options (popularity / rating / title) — API currently returns RRF order only.
7. Pagination or cursor — API is limit-only today.
8. Book detail page aggregating all sources, covers, and access links.
9. Trending from the local Project Gutenberg catalog (downloads are already in SQLite).

### Slice 3 — bigger bets (test each before building)

10. Save / shelve books (needs local storage or user identity).
11. "Similar books" via subject overlap.
12. Full-text search inside books (Gutenberg / Internet Archive) — heavy.

---

## 5. Desk research — Reddit evidence (2026-10-02)

Pre-interview evidence sweep. Method: site-restricted web searches over
reddit.com (thread bodies were not readable — Reddit blocks fetchers — so all
evidence is snippet/search-summary level). Not a substitute for interviews;
it sharpens the frame and the odds.

### What the threads show

| Hypothesized signal | Evidence found | Strength |
|---|---|---|
| **Frequency: people repeatedly hunt for books on a topic** | The same question recurs across communities and years: r/books ("What websites do you use for book recommendations?", "How can I find books on topics I don't know about?"), r/history, r/nonfictionbookclub — plus two entire subreddits (r/suggestmeabook, r/booksuggestions) existing to answer it | Strong |
| **Workaround: multi-tool hunt, exactly as hypothesized** | Goodreads dominates, supplemented by fivebooks.com (expert lists), Googling "best books on [topic]", citation trails, and asking subreddits/librarians. Quality tools and access tools are entirely separate | Strong |
| **Pain: Goodreads discovery is resented** | Recurring "Goodreads alternatives" threads (r/books 2021 & 2025, r/Fantasy "Why isn't there a better alternative?"); complaints: outdated search, stagnation, Amazon ownership. A graveyard of scrappy competitors (StoryGraph, Literal, margins, PageBound) proves people keep trying to escape | Strong — but note: those competitors all attack tracking/social, none attack "readable free now" |
| **"Readable free, legally, now" matters** | PG is shared as a *tip* in multiple PSA/YSK threads ("Don't forget to check Project Gutenberg…") — free sources are treated as hidden gems, i.e. an awareness/discovery gap. The r/Piracy books megathread lists *legal* options (Gutenberg, LibriVox) — when the legal path is fragmented, demand drifts to piracy. Librarians hand-build aggregate lists (r/Libraries "Liberation Library": IA + DOAB + DOAJ combined manually) | Strong — someone is manually doing Lux's job today |

### The sharpest finding

**The gap in the market is not another Goodreads** (tracking/social/recommendations
are saturated and network-effects-bound). It is the missing bridge between the
*quality* tools (Goodreads, fivebooks — "what's good") and the *access* tools
(Gutenberg, IA, DOAB — "what can I read now"): **"find the best books on a
topic that I can legally start reading for free, right now."** Every ingredient
of the hypothesized pain is publicly, repeatedly complained about — and at
least one librarian is manually aggregating the exact catalogs Lux federates.

### Effect on the assessment (§2)

Q1 (real/frequent): 3 → **4** — perennial cross-community threads.
Q2 (hated workaround): 3 → **4** — loud, recurring Goodreads-discovery
complaints + visible tool fragmentation.
Total: 17/25 → **19/25** (worth a discovery cycle, now with external evidence).

### What desk research cannot answer

Whether users *switch* — that only the prototype test shows (pass/fail
criteria in §3 stand). Snippet-level evidence also over-represents vocal
Redditors; the interview screener still guards for that.

### Sources

Reddit threads (via search summaries): [r/books: What websites do you use for book recommendations?](https://www.reddit.com/r/books/comments/txqn1r/what_websites_do_you_use_for_book_recommendations) ·
[r/books: How can I find books on topics I don't know about?](https://www.reddit.com/r/books/comments/7l8u7f/how_can_i_find_books_on_topics_i_dont_know_about) ·
[r/nonfictionbookclub: How do you find nonfiction books?](https://www.reddit.com/r/nonfictionbookclub/comments/o9gvfk/how_do_you_find_nonfiction_books_youre_interested) ·
[r/books: GoodReads Alternatives?](https://www.reddit.com/r/books/comments/1jhf0bh/goodreads_alternatives) ·
[r/Fantasy: Why isn't there a better alternative?](https://www.reddit.com/r/Fantasy/comments/16breoe/goodreads_is_the_most_popular_book_website_so_why) ·
[r/books: What would make you switch from Goodreads?](https://www.reddit.com/r/books/comments/k4bwck/what_would_make_you_switch_from_goodreads_to) ·
[r/books: Don't forget to check Project Gutenberg](https://www.reddit.com/r/books/comments/3916s9/dont_forget_to_check_project_gutenberg_for_free) ·
[r/YouShouldKnow: Project Gutenberg has 57,000+ books](https://www.reddit.com/r/YouShouldKnow/comments/15g1ff5/ysk_that_project_gutenberg_has_an_online_library) ·
[r/Piracy wiki: Books & Audiobooks megathread](https://www.reddit.com/r/Piracy/wiki/megathread/books) ·
[r/Libraries: Building a Liberation Library (OA/CC/PD)](https://www.reddit.com/r/Libraries/comments/1pqceim/building_a_liberation_library_oa_cc_pd) ·
[r/AskReddit: Free textbooks](https://www.reddit.com/r/AskReddit/comments/1w4tuwa/what_are_some_websites_that_have_free_textbooks)
