# Forage CLI Usage

`forage` is a provider-aware research retrieval CLI. It exposes stable capabilities and hides provider fallback, quota cooldown, and credential lookup behind the router.

## Agent-First Commands

These are the preferred commands for agents. They return evidence records rather than forcing the caller to manually chain `search`, `fetch`, and `extract`.

```powershell
forage gather "latest OpenAI model pricing" --json
forage retrieve "https://example.com" --json
forage retrieve "10.1038/nature12373" --json
forage brief "Browserbase search API docs" --format markdown
```

Use primitives such as `search web`, `fetch`, and `extract` when you need explicit provider debugging or a single low-level capability.

### `gather`

Collect usable evidence records for a query:

```powershell
forage gather "Tavily API rate limits" --mode auto --limit 8 --fetch 5 --json
forage gather "AI regulation" --mode mixed --limit 9 --fetch 6 --save-pack --json
```

Useful flags:

```powershell
--mode auto|web|news|scholar|mixed
--limit 8
--fetch 5
--cache auto|refresh|only
--max-chars 4000
--save-pack
--explain-routing
```

JSON `data` contains:

```json
{
  "query": "Tavily API rate limits",
  "mode": "auto",
  "records": [
    {
      "title": "...",
      "url": "...",
      "source_domain": "...",
      "search_provider": "brave",
      "fetch_provider": "jina",
      "text": "...",
      "retrieved_at": "...",
      "content_hash": "...",
      "quality_score": 0.95,
      "cache_status": "miss"
    }
  ],
  "evidence_pack_path": ".forage/evidence/<id>.json"
}
```

### `retrieve`

Route arbitrary input without requiring the caller to classify it:

```powershell
forage retrieve "Browserbase free tier limits" --json
forage retrieve "https://example.com" --json
forage retrieve "10.1038/nature12373" --json
forage retrieve "0000-0002-1825-0097" --json
```

Useful flags:

```powershell
--kind auto|url|doi|paper|author|query
--cache auto|refresh|only
--max-chars 4000
--save-pack
--explain-routing
```

JSON `data` contains `input`, inferred or forced `kind`, `result`, normalized `records`, optional `routing`, and optional `evidence_pack_path`.

### `brief`

Render gathered evidence as compact source-numbered context blocks:

```powershell
forage brief "Browserbase search API docs" --format markdown
forage brief "OpenAlex API rate limits" --format context --max-chars 1000
forage brief "machine learning benchmarks" --mode mixed --json
```

Useful flags:

```powershell
--format markdown|json|context
--mode auto|web|news|scholar|mixed
--limit 6
--fetch 4
--cache auto|refresh|only
--max-chars 1200
--explain-routing
```

Markdown/context output is designed to be pasted directly into an LLM context window. JSON output includes both `records` and the rendered `context`.

## Setup

Create the global config:

```powershell
forage config init
forage config repair
```

Interactive setup stores API keys in the OS keychain:

```powershell
forage setup
```

Non-interactive credential setup:

```powershell
"<api-key>" | forage credentials set brave --value-stdin
forage credentials set brave --field search_api_key --from-env BRAVE_SEARCH_API_KEY
forage credentials set brave --field answers_api_key --from-env BRAVE_ANSWERS_API_KEY
forage credentials list
forage credentials check brave
forage credentials remove brave
```

Secrets are not written to `.forage/config.yaml`.

For local validation without manually exporting variables:

```powershell
Copy-Item .env.example .env
notepad .env
forage providers doctor --all --json
```

`.env` is repo-local, gitignored, and loaded before credential checks. Existing process environment variables win over values from `.env`.

No credentials are needed for Hacker News, Crossref, arXiv, DataCite, Wikidata, Internet Archive, Common Crawl, Forem, DOAJ, or Europe PMC. Semantic Scholar and GDELT are registered as metadata-only until their public endpoints are reliable from the Windows release environment.

Blogger, WordPress API endpoints, Reddit, Google Custom Search, Diffbot, ScraperAPI, and Europeana are intentionally excluded from the v0.1 provider setup.

See `PROVIDER_STATUS.md` for the current release-verification target and metadata-only provider list.

## Health and Quota

```powershell
forage providers list
forage providers doctor
forage providers doctor jina
forage providers doctor --capability search.web
forage providers doctor --all
forage providers quota
forage providers quota --provider brave
forage providers quota --preflight --provider openalex
forage providers quota --reset-local brave
```

Use `--json` for agent-safe output and `--verbose` or `--explain-routing` for provider attempts.

## Advanced Search Primitives

```powershell
forage search web "query" --limit 10 --json
forage search news "query" --providers gdelt,guardian --json
forage search scholar "query" --providers openalex,crossref,arxiv --json
forage search platform "query" --providers hackernews,forem --json
```

Useful flags:

```powershell
--limit 20
--site example.com
--providers brave,jina,tavily
--exclude-provider brave
--cache auto|refresh|only
--explain-routing
```

## Advanced Fetch, Extract, Render, Crawl

```powershell
forage fetch https://example.com --providers direct --json
forage extract https://example.com --json
Get-Content urls.txt | forage extract --stdin --json
forage render https://example.com --json
forage map https://example.com --max-pages 25 --json
forage crawl https://example.com --max-pages 10 --json
```

Hosted extraction providers are preferred when configured. Direct fetch is the local fallback.

## Scholar, Citations, Archives, Corpora

```powershell
forage enrich doi 10.1038/nature12373 --json
forage enrich paper 10.1038/nature12373 --json
forage citations 10.1038/nature12373 --json
forage archive lookup https://example.com --json
forage corpus commoncrawl --json
forage corpus gdelt "climate" --json
```

## Evidence Packs

```powershell
forage research-pack "machine learning" --json
forage evidence inspect .forage/evidence/<id>.json --json
```

You can also create a pack from JSON or JSONL stdin:

```powershell
forage search scholar "machine learning" --json | forage evidence create --query "machine learning" --json
```

## Cache

```powershell
forage cache status
forage cache clear
```

Cache state lives under `.forage/state.db`.
