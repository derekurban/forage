# Provider Status

Forage treats provider status conservatively. A provider is not release-verified until it has mocked adapter coverage for success, auth failure, quota/rate-limit failure, malformed responses, timeout behavior, and an optional gated live smoke test.

## Locally Verified For v0.1.0

These providers have mocked adapter coverage and passed local Windows smoke checks with the repo-local `.env` on 2026-05-22:

| Capability | Providers |
| --- | --- |
| Web search | Brave, Jina, Browserbase, Tavily, Exa, SerpApi, serpstack |
| Fetch/extract | Jina Reader, Browserbase Fetch, Firecrawl, ScrapingAnt, Direct HTTP |
| News | Brave News, Guardian, GNews, NewsAPI, Currents, Mediastack, World News API |
| Platform | Hacker News, Forem |
| Scholar | OpenAlex, Crossref, arXiv, PubMed, DataCite, Europe PMC, DOAJ |
| Archive/corpus | Internet Archive, Common Crawl |

Provider verification means success, auth/rate-limit/server-failure handling in tests, and at least one local live smoke command where credentials are required.

## Metadata Or Credential Only

These providers remain registered for setup and future adapters, but should not be treated as production live paths until promoted by tests and smoke checks:

| Provider | Current role |
| --- | --- |
| Apify | Metadata-only crawl/extract/render provider |
| Browserless | Metadata-only render provider |
| GDELT | Metadata-only news/archive provider; public endpoint is slow/unreliable from the Windows release environment |
| Semantic Scholar | Adapter exists, but unauthenticated public endpoint returned 429 during local smoke checks |
| Wikidata | Metadata-only identity graph provider |
| OpenCitations | Metadata-only citation provider |
| ORCID | Metadata-only identity provider |
| Unpaywall | Metadata-only open-access provider; requires `UNPAYWALL_EMAIL` when implemented |

## Removed From Active Scope

Forage intentionally excludes Reddit, Google Custom Search, Blogger, WordPress API endpoints, Diffbot, ScraperAPI, and Europeana from default setup and routing.

## Local Verification

After filling `.env`, use:

```powershell
go test ./...
go run ./cmd/forage providers doctor --all --json
go run ./cmd/forage search web "openai" --limit 3 --cache refresh --explain-routing --json
go run ./cmd/forage search news "climate" --limit 3 --cache refresh --explain-routing --json
go run ./cmd/forage search scholar "machine learning" --limit 3 --cache refresh --json
go run ./cmd/forage fetch https://example.com --cache refresh --json
```

## Quota Tracking

Forage tracks quota in three tiers:

| Tracking mode | Providers | What Forage can do |
| --- | --- | --- |
| Response headers | Brave, Browserbase, OpenAlex, PubMed, Guardian, World News API, Crossref when headers appear | Persist observed limit, remaining, reset, retry, and provider status |
| Provider endpoint | OpenAlex | Preflight/check account quota with `forage providers quota --preflight --provider openalex` |
| Manual/inferred | Tavily, Exa, Firecrawl, Jina, GNews, NewsAPI, Currents, Mediastack, SerpApi, serpstack, Semantic Scholar, public/free APIs | Track local attempts, successes, 429s, failures, and documented manual limits; skip providers when local request budget is exhausted |

Sources used for quota behavior include Brave rate-limit headers, Browserbase `RateLimit-*` headers, OpenAlex `X-RateLimit-*` headers and `/rate-limit`, Exa QPS docs, Firecrawl 429 behavior, World News API quota headers, Serpstack usage-limit error bodies, and Semantic Scholar public/authenticated rate notes.
