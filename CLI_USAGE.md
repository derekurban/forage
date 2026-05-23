# CLI Usage

`forage` is a complementary retrieval CLI for agents. Use native web search for broad discovery. Use Forage when you need clean extracted page text, scholarly metadata/citations, or archive verification.

All commands support stable JSON envelopes with:

```json
{
  "ok": true,
  "command": "forage ...",
  "generated_at": "...",
  "data": {},
  "diagnostics": null,
  "error": null
}
```

## `extract`

Extract one URL or a file of URLs into normalized documents.

```powershell
forage extract "https://example.com/article" --json
forage extract urls.txt --jsonl
Get-Content urls.txt | forage extract --stdin --jsonl
```

Flags:

```text
--cache              auto, refresh, or only
--explain-routing    include provider routing diagnostics
```

Returned records include URL, title when available, Markdown/plain text, provider, extraction method, retrieved timestamp, content hash, quality score, and cache status.

Use this after native search has found candidate sources.

## `scholar`

Search and enrich scholarly material through scholarly APIs.

```powershell
forage scholar "memory consolidation transformer models" --limit 10 --json
forage scholar doi "10.1038/nature12373" --json
forage scholar paper "10.1038/nature12373" --json
forage scholar citations "10.1038/nature12373" --json
forage scholar doi "10.1038/nature12373" --raw --json
```

Flags:

```text
--limit              maximum scholarly records for query search
--cache              auto, refresh, or only
--explain-routing    include provider routing diagnostics
--raw                include raw provider payloads in JSON output
```

Use this when native search is not enough for DOI metadata, paper identifiers, citation expansion, PubMed/arXiv-style records, open-access lookup, or dataset metadata. Default output is compact and normalized for agents; `--raw` keeps provider-specific payloads available when you need fields outside the normalized contract.

## `archive`

Check historical/source availability for a URL.

```powershell
forage archive "https://example.com/article" --json
forage archive "https://example.com/article" --explain-routing --json
```

Flags:

```text
--limit              maximum archive records
--cache              auto, refresh, or only
--explain-routing    include provider routing diagnostics
```

Returned records can include Wayback availability and Common Crawl index metadata. Use this to verify disappeared pages, old sources, and historical web presence.

## Setup And Health

Initialize repo-local config:

```powershell
forage config init
```

Load local validation credentials:

```powershell
Copy-Item .env.example .env
```

Store stable credentials in the OS keychain if desired:

```powershell
forage credentials list
forage credentials set jina --from-env JINA_API_KEY
forage credentials check jina
forage credentials remove jina
```

Check provider readiness:

```powershell
forage providers list
forage providers doctor --all --json
forage providers quota
forage providers quota --preflight --provider openalex
```

Repair older configs after provider trimming:

```powershell
forage config repair
```

## Credential Fields

`.env.example` intentionally contains only credentials relevant to the three primitives:

```dotenv
FORAGE_CONTACT_EMAIL=
UNPAYWALL_EMAIL=
JINA_API_KEY=
FIRECRAWL_API_KEY=
BROWSERBASE_API_KEY=
SCRAPINGANT_API_KEY=
OPENALEX_API_KEY=
NCBI_API_KEY=
OPENCITATIONS_ACCESS_TOKEN=
```

No credentials are needed for Crossref, arXiv, DataCite, DOAJ, Europe PMC, Semantic Scholar, Internet Archive, Common Crawl, or direct HTTP extraction. `UNPAYWALL_EMAIL` is a required contact email for Unpaywall requests, not a secret.

## Provider Scope

Supported product capabilities:

- Extraction: Jina Reader, Browserbase Fetch, Firecrawl, ScrapingAnt, Direct HTTP
- Scholar: OpenAlex, Crossref, arXiv, PubMed/NCBI, DataCite, DOAJ, Europe PMC, Semantic Scholar
- DOI/open-access/citation enrichment: Crossref, OpenAlex, Unpaywall, OpenCitations, Semantic Scholar
- Archive: Internet Archive, Common Crawl

Removed from primary support:

- Generic web search, news search, social/platform search, generic crawling, generic rendering, and report-writing commands.
- Brave, Tavily, Exa, SerpApi, serpstack, Guardian, Currents, NewsAPI, GNews, Mediastack, World News API, GDELT, Hacker News, Reddit, Forem, Blogger, WordPress, Google CSE, Apify, Browserless, ORCID, Wikidata, Diffbot, ScraperAPI, and Europeana.

## Exit Behavior

If no eligible provider can satisfy a primitive, Forage returns a structured error with an exit code and setup hint. Intermediate provider failures stay hidden unless `--verbose`, `--explain-routing`, doctor, quota, or JSON diagnostics are requested.
