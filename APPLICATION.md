# Forage Application Specification

## Purpose

`forage` is a Go CLI and library for research-oriented web retrieval.

Its purpose is to give deep research agents, shell scripts, and human operators a dependable way to turn queries, URLs, DOIs, domains, and source identifiers into normalized evidence records.

`forage` is not the research agent itself. It is the retrieval and evidence I/O layer underneath an agent.

The application owns provider setup, provider routing, quota handling, fallback behavior, extraction, normalization, caching, and provenance. Higher-level agents own planning, reasoning, synthesis, and final report writing.

## Primary Product Promise

Given a research capability request, `forage` should use the best configured provider path available, gracefully fall back when providers fail or exhaust quota, and return either normalized evidence or a structured final failure.

The caller should not need to manage provider rate limits or understand provider-specific APIs.

## Intended Users

| User | Need |
|---|---|
| Deep research agents | Stable tools for search, fetch, extraction, enrichment, caching, and provenance. |
| Shell users | Composable commands that work with pipes, files, `jq`, `xargs`, `sqlite3`, and other CLI tools. |
| Application developers | A Go library with stable primitives and normalized result types. |
| Operators | Visibility into provider setup, quota state, cache state, and routing diagnostics. |

## Scope

### In scope

`forage` provides primitives for:

- Web search.
- News search.
- Scholarly search.
- Platform-specific discovery.
- URL fetching.
- Article/document extraction.
- JS-rendered page retrieval.
- Site crawling within configured limits.
- DOI, citation, author, organization, and entity enrichment.
- Archive lookup.
- Result normalization.
- Deduplication.
- Local caching.
- Provider health checks.
- Quota and rate-limit tracking.
- Graceful provider fallback.
- Provenance and routing diagnostics.

### Out of scope

`forage` does not provide:

- Autonomous research planning.
- Long-running agent reasoning loops.
- Final answer synthesis.
- Report writing as a core feature.
- LLM-dependent orchestration as a required path.
- A full vector database.
- A citation manager.
- A browser UI.
- A web application.
- A general scraping DSL.
- Unbounded crawling.
- Provider-specific APIs as the primary user interface.

Some of these may be built above `forage`, but they should not be core responsibilities of this application.

## Application Boundary

The central boundary is:

> Callers request capabilities. `forage` handles providers.

For example, a caller should ask for:

```text
search.web
extract.article
search.scholar
enrich.citations
archive.lookup
```

The caller should not need to decide:

```text
try Brave, then Exa, then Tavily, then SerpApi, unless Tavily credits are exhausted, unless Brave is cooling down, unless the cache is fresh enough
```

That provider choreography belongs inside `forage`.

## Exposed Surface

The exposed surface should be stable, capability-oriented, and easy to compose.

### CLI commands

Representative command surface:

```text
forage search web "query"
forage search news "query"
forage search scholar "query"
forage fetch https://example.com/article
forage extract urls.txt
forage crawl https://example.com
forage enrich doi 10.1234/example
forage archive lookup https://example.com/article
forage providers list
forage providers doctor
forage providers quota
forage config init
```

### Output formats

Supported output modes should include:

| Format | Purpose |
|---|---|
| Human table | Manual inspection and debugging. |
| JSON | Agent and application integration. |
| JSONL | Streaming and batch pipelines. |
| Markdown | Human-readable evidence bundles. |
| SQLite cache | Local replay, dedupe, and provenance. |

### Global behavior flags

Common flags should include:

| Flag | Purpose |
|---|---|
| `--json` | Emit one structured JSON response. |
| `--jsonl` | Emit newline-delimited records. |
| `--profile free` | Use only recurring-free and local providers. |
| `--profile paid-ok` | Allow paid providers within configured budget. |
| `--policy strict` | Apply conservative compliance and robots handling. |
| `--policy research` | Use research-oriented provider policy. |
| `--max-cost 0` | Disallow paid spend. |
| `--max-requests N` | Bound provider fanout. |
| `--providers a,b,c` | Pin providers for reproducibility. |
| `--exclude-provider x` | Skip a provider for one invocation. |
| `--cache only` | Do not call network providers. |
| `--cache refresh` | Prefer cache, but refresh stale records. |
| `--explain-routing` | Include provider routing diagnostics. |

### Stable result records

Callers should receive normalized records such as:

- `SearchResult`
- `ExtractedDocument`
- `RenderedPage`
- `WorkResult`
- `CitationGraph`
- `ArchiveRecord`
- `EntityRecord`
- `EvidencePack`

Provider-specific raw payloads may be attached for debugging, but the primary schema should remain provider-neutral.

## Encapsulated Internals

The following concerns should be handled internally and not pushed onto the agent or normal CLI caller.

### Provider routing

`forage` decides which configured providers can satisfy a capability request.

Routing inputs include:

- Requested capability.
- Available credentials.
- Provider health.
- Remaining quota.
- Cooldown state.
- Cache freshness.
- Cost profile.
- Compliance profile.
- Freshness requirement.
- Content type.
- Historical provider quality.

### Rate-limit and quota handling

The caller should not normally see provider rate-limit failures.

Provider adapters and the router should:

- Track known provider limits.
- Track observed usage.
- Parse rate-limit headers where available.
- Parse provider-specific error bodies.
- Respect `Retry-After` and reset hints.
- Apply cooldowns.
- Move exhausted providers out of routing.
- Fall back to the next eligible provider.
- Return a final failure only when all configured routes are unavailable.

### Provider-specific APIs

Provider-specific request and response details should remain inside provider adapters.

The public CLI and library should not expose provider-specific fields as required inputs unless there is a clear advanced-use case.

### Extraction escalation

The caller should be able to ask for extracted content from a URL. Internally, `forage` may choose:

1. Cache hit.
2. Plain HTTP fetch.
3. Hosted reader API.
4. Local readability extraction.
5. Rendered browser fetch.
6. Structured extraction provider.

This escalation should be automatic when enabled by policy and budget.

### Deduplication and normalization

Provider-specific results should be normalized and merged internally using:

- Canonical URL.
- DOI.
- PMID.
- arXiv ID.
- OpenAlex ID.
- Semantic Scholar ID.
- Normalized title.
- Content hash.
- Source domain.

The caller should receive merged evidence candidates rather than raw duplicate provider records by default.

## Provider Limit Registry

`forage` needs an itemized provider limit registry.

This registry should describe both documented limits and runtime-observable limit signals.

### Required provider limit fields

| Field | Meaning |
|---|---|
| `provider` | Stable provider identifier. |
| `capabilities` | Supported capabilities such as `search.web`, `extract.article`, `render.browser`. |
| `auth_type` | `none`, `api_key`, `oauth`, `custom`. |
| `limit_type` | `rps`, `rpm`, `daily`, `monthly`, `credits`, `concurrency`, `browser_time`, `shared_pool`, `unknown`. |
| `limit_scope` | `key`, `account`, `ip`, `oauth_client`, `provider_global`, `unknown`. |
| `reset_model` | `fixed_window`, `rolling_window`, `header_reported`, `manual`, `unknown`. |
| `request_cost_model` | Per request, per endpoint, per credit, per token, per browser unit, or unknown. |
| `headers` | Headers that expose remaining quota, reset time, retry time, or request cost. |
| `status_codes` | Provider-specific rate-limit, quota, auth, and block signals. |
| `body_fields` | Response body fields that expose quota or errors. |
| `documented_limits` | Human-readable source-of-truth limits. |
| `confidence` | `documented`, `observed`, `configured`, `unknown`. |
| `fallback_class` | How the router should respond when this provider is unavailable. |

### Runtime quota state

At runtime, each provider should maintain local state:

| State | Meaning |
|---|---|
| `healthy` | Eligible for normal routing. |
| `conserve` | Usable, but below reserve threshold. |
| `cooldown` | Temporarily unavailable due to short-term rate limiting. |
| `exhausted` | Quota unavailable until known or estimated reset. |
| `degraded` | Repeated failures or low quality. |
| `disabled` | Missing credentials, policy mismatch, or user-disabled. |

The router uses this state to choose fallbacks. Normal callers should not need to inspect it.

## Failure Behavior

`forage` should hide intermediate provider failures when fallback succeeds.

When every configured path fails, the final error should be structured, actionable, and machine-readable.

Example final failure:

```json
{
  "error": {
    "code": "capability_exhausted",
    "capability": "search.web",
    "message": "No configured web search providers are currently available.",
    "retry_after": "2026-05-23T00:00:00Z",
    "providers": [
      {
        "name": "brave",
        "status": "quota_exhausted",
        "retry_after": "2026-06-01T00:00:00Z"
      },
      {
        "name": "tavily",
        "status": "cooldown",
        "retry_after": "2026-05-22T18:10:00Z"
      },
      {
        "name": "serpapi",
        "status": "disabled",
        "reason": "auth_missing"
      }
    ]
  }
}
```

### Exit codes

Suggested exit codes:

| Code | Meaning |
|---:|---|
| `0` | Success. |
| `1` | General failure. |
| `2` | Invalid arguments or config. |
| `3` | Capability exhausted after fallback. |
| `4` | No provider configured for requested capability. |
| `5` | Authentication/setup failure. |
| `6` | Compliance or policy block. |
| `7` | Cache miss in `--cache only` mode. |
| `8` | Partial success with some failed records. |

## Caching

Caching is a core application feature, not an optimization afterthought.

The cache should store:

- Normalized records.
- Provider name.
- Raw response reference.
- Request parameters.
- Retrieval timestamp.
- Content hash.
- Canonical identifiers.
- Extraction method.
- Quality score.
- Compliance flags.
- Quota observations.
- Negative outcomes with short TTLs.

Cache behavior should support:

- Cache-first retrieval.
- Cache-only replay.
- Refresh stale records.
- Deduplication across providers.
- Reproducible agent runs.

## Compliance and Policy

Compliance decisions should be first-class routing inputs.

Policy should account for:

- Provider terms.
- Commercial-use restrictions.
- Robots handling.
- User-agent requirements.
- OAuth/API terms.
- Content retention restrictions.
- Deleted/private/protected content.
- Paywall and access status.
- Crawl limits.

The application should avoid treating "technically fetchable" as equivalent to "allowed for this workflow."

## Configuration

`v0.1.0` uses one repo-local global config file:

```text
.forage/config.yaml
```

There is no user-home config, project/user precedence, or scoped permission model in this release. CLI flags can override one invocation, and environment variables can supply credentials, but the only persisted non-secret configuration lives in `.forage/config.yaml`.

The CLI should remain useful with partial setup:

| Setup level | Expected behavior |
|---|---|
| No keys | Local fetch/extraction plus public/no-key providers where allowed; web search fails clearly if no web-search key is configured. |
| Easy API keys | Better search, extraction, and scholarly coverage. |
| OAuth configured | Reddit, WordPress.com, ORCID, and other account-based providers. |
| Browser providers configured | JS-heavy extraction, screenshots, and hard pages. |
| Paid providers enabled | Higher reliability and scale within explicit budget limits. |

## Library Boundary

The Go library should expose stable primitives and normalized types.

It should not require callers to instantiate provider-specific clients for normal use.

Recommended public package responsibilities:

- Construct a client from config.
- Execute capability requests.
- Return normalized records.
- Return structured errors.
- Allow optional routing diagnostics.

Provider adapters, quota parsers, fallback chains, and raw API details should remain internal unless there is a strong reason to expose them.

## Example Shell Composition

Search, extract, and store evidence:

```bash
forage search web "AI browser agents" --jsonl \
  | jq -r '.url' \
  | forage extract --stdin --jsonl \
  > evidence.jsonl
```

Use cache-only replay:

```bash
forage search web "AI browser agents" --cache only --json
```

Show provider state for an operator:

```bash
forage providers quota
```

Explain routing for debugging:

```bash
forage search news "example topic" --explain-routing --json
```

## Design Principles

1. Expose capabilities, not provider choreography.
2. Hide rate-limit fallback during successful runs.
3. Return structured final failure when all routes are unavailable.
4. Normalize results early.
5. Preserve raw provider data for audit and debugging.
6. Make cache and provenance core to every workflow.
7. Make policy and compliance explicit.
8. Keep output composable with normal CLI tooling.
9. Make partial setup useful.
10. Keep synthesis and reasoning above the CLI, not inside it.
