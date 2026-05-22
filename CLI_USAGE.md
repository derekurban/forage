# Forage CLI Usage

`forage` is a provider-aware research retrieval CLI. It exposes stable capabilities and hides provider fallback, quota cooldown, and credential lookup behind the router.

## Setup

Create the global config:

```powershell
forage config init
```

Interactive setup stores API keys in the OS keychain:

```powershell
forage setup
```

Non-interactive credential setup:

```powershell
"<api-key>" | forage credentials set brave --value-stdin
forage credentials set brave --from-env BRAVE_API_KEY
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

No credentials are needed for Hacker News, GDELT, Semantic Scholar public endpoints, Crossref, arXiv, DataCite, Wikidata, Internet Archive, Common Crawl, Forem, DOAJ, or Europe PMC.

Blogger, WordPress API endpoints, Reddit, Google Custom Search, Diffbot, and ScraperAPI are intentionally excluded from the v0.1 provider setup.

## Health and Quota

```powershell
forage providers list
forage providers doctor
forage providers doctor jina
forage providers doctor --capability search.web
forage providers doctor --all
forage providers quota
forage providers quota --provider brave
forage providers quota --reset-local brave
```

Use `--json` for agent-safe output and `--verbose` or `--explain-routing` for provider attempts.

## Search

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

## Fetch, Extract, Render, Crawl

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

You can also create a pack from JSON stdin:

```powershell
forage search scholar "machine learning" --json | forage evidence create --query "machine learning" --json
```

## Cache

```powershell
forage cache status
forage cache clear
```

Cache state lives under `.forage/state.db`.
