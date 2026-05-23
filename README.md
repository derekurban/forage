# Forage

Forage is a Windows-first Go CLI that complements native web search. It is not trying to be a better generic search engine. Its job is to handle the research primitives that native search tools do not expose reliably:

- `extract`: turn URLs or URL lists into clean Markdown/text evidence.
- `scholar`: query and enrich scholarly records, DOIs, and citations.
- `archive`: check historical/source availability through archive APIs.

Native web search should remain the first-pass discovery layer. Forage is the follow-up retrieval layer for extraction, academic metadata, citation/open-access lookup, and historical verification.

## Install

Download the Windows zip from [GitHub Releases](https://github.com/derekurban/forage/releases), or install with Go:

```powershell
go install github.com/derekurban/forage/cmd/forage@latest
```

Then initialize repo-local state:

```powershell
forage config init
```

Forage uses `.forage/config.yaml` in the current repo. It does not use user-home config or per-project permission scopes.

## Core Commands

Extract a discovered URL into clean text:

```powershell
forage extract "https://example.com/article" --json
forage extract urls.txt --jsonl
```

Search or enrich scholarly material:

```powershell
forage scholar "memory consolidation transformer models" --json
forage scholar doi "10.1038/nature12373" --json
forage scholar paper "10.1038/nature12373" --json
forage scholar citations "10.1038/nature12373" --json
forage scholar doi "10.1038/nature12373" --raw --json
```

Check archive availability for a URL:

```powershell
forage archive "https://example.com/article" --json
forage archive "https://example.com/article" --limit 3 --json
```

## Credentials

For local validation, copy the template and fill in only the credentials relevant to extraction and scholarly enrichment:

```powershell
Copy-Item .env.example .env
```

`.env` is gitignored and loaded before credential checks. Existing process environment variables win over `.env`. OS keychain storage is still supported for longer-lived local setup:

```powershell
forage credentials list
forage credentials set jina --from-env JINA_API_KEY
forage credentials set firecrawl --from-env FIRECRAWL_API_KEY
```

The reduced credential surface is:

- Extraction: `JINA_API_KEY`, `FIRECRAWL_API_KEY`, `BROWSERBASE_API_KEY`, `SCRAPINGANT_API_KEY`
- Scholarly/citations: `OPENALEX_API_KEY`, `NCBI_API_KEY`, `OPENCITATIONS_ACCESS_TOKEN`, `UNPAYWALL_EMAIL`
- Contact identity: `FORAGE_CONTACT_EMAIL`

Most scholarly/archive providers need no key: Crossref, arXiv, DataCite, DOAJ, Europe PMC, Semantic Scholar, Internet Archive, and Common Crawl. Unpaywall requires `UNPAYWALL_EMAIL` as a contact email, not a secret.

## Operational Commands

These commands exist to keep the three primitives healthy:

```powershell
forage providers list
forage providers doctor --all
forage providers quota
forage providers quota --preflight --provider openalex
forage cache status
forage config repair
forage version
```

Use `--json` for agent-safe output. Use `--verbose` or `--explain-routing` when you need provider diagnostics.

## Provider Scope

Forage intentionally supports only providers that serve `extract`, `scholar`, or `archive`.

Kept:

- Extraction: Jina Reader, Browserbase Fetch, Firecrawl, ScrapingAnt, Direct HTTP
- Scholar: OpenAlex, Crossref, arXiv, PubMed/NCBI, DataCite, DOAJ, Europe PMC, Semantic Scholar
- Enrichment/citations: Unpaywall, OpenCitations
- Archive: Internet Archive, Common Crawl

Removed from the product surface:

- Generic web search: Brave, Tavily, Exa, SerpApi, serpstack, Google CSE
- News/event feeds: Guardian, Currents, NewsAPI, GNews, Mediastack, World News API, GDELT
- Platform/blog/social sources: Hacker News, Reddit, Forem, Blogger, WordPress
- Broad render/crawl vendors without a current extraction role: Apify, Browserless
- Identity/entity extras: ORCID, Wikidata
- Paid/trial/enterprise-only providers: Diffbot, ScraperAPI, Europeana

## Development

```powershell
go test ./...
go build ./cmd/forage
```

Live provider tests should stay opt-in and gated by `FORAGE_LIVE_TESTS=1` plus provider-specific credentials.
