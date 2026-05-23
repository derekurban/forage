# Provider Status

Forage now has a narrow provider scope. Providers are included only when they support one of the complementary primitives: `extract`, `scholar`, or `archive`.

## Live-Supported

These providers are part of default routing and have adapter coverage in the current codebase:

| Capability | Providers |
| --- | --- |
| Extract | Jina Reader, Browserbase Fetch, Firecrawl, ScrapingAnt, Direct HTTP |
| Scholar search | OpenAlex, Crossref, arXiv, PubMed/NCBI, DataCite, DOAJ, Europe PMC, Semantic Scholar |
| DOI/paper enrichment | OpenAlex, Unpaywall, Crossref, Semantic Scholar |
| Citation lookup | OpenCitations, OpenAlex, Semantic Scholar, Crossref fallback metadata |
| Archive lookup | Internet Archive, Common Crawl |

## Raw Payload Policy

Scholarly commands return compact normalized records by default. Use `--raw` on `scholar`, `scholar doi`, `scholar paper`, or `scholar citations` when an agent needs provider-specific fields outside the normalized contract.

## Removed

These providers are intentionally removed from setup, default routing, and product documentation because they do not serve the reduced primitive set:

| Category | Providers |
| --- | --- |
| Generic web search | Brave, Tavily, Exa, SerpApi, serpstack, Google CSE |
| News/event feeds | Guardian, Currents, NewsAPI, GNews, Mediastack, World News API, GDELT |
| Platform/blog/social | Hacker News, Reddit, Forem, Blogger, WordPress |
| Broad render/crawl | Apify, Browserless |
| Identity/entity extras | ORCID, Wikidata |
| Paid/trial/enterprise-only | Diffbot, ScraperAPI, Europeana |

## Validation Commands

```powershell
go run ./cmd/forage config repair
go run ./cmd/forage providers list
go run ./cmd/forage providers doctor --all --json
go run ./cmd/forage extract https://example.com --cache refresh --json
go run ./cmd/forage scholar "machine learning" --limit 3 --cache refresh --json
go run ./cmd/forage scholar doi "10.1038/nature12373" --cache refresh --json
go run ./cmd/forage archive https://example.com --cache refresh --json
```

Optional live tests should remain gated behind `FORAGE_LIVE_TESTS=1` plus provider-specific environment variables.

## Quota Tracking

| Mode | Providers | Behavior |
| --- | --- | --- |
| Observed headers | Browserbase, OpenAlex, PubMed, Crossref where headers appear | Persist observed limit, remaining, reset, retry, and provider status |
| Provider endpoint | OpenAlex | Refresh with `forage providers quota --preflight --provider openalex` |
| Manual/inferred | Jina, Firecrawl, ScrapingAnt, Semantic Scholar, public scholarly/archive APIs | Track local attempts, successes, 429s, failures, and documented manual limits |

Removed providers are not tracked for quota in the reduced product surface.
