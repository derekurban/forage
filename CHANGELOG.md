# Changelog

## v0.1.4

- Sanitized OpenAlex quota preflight observations so provider diagnostic output never stores or emits API key fields from provider response bodies.

## v0.1.3

- Routed archive, enrichment, citations, corpus, render, crawl, and map commands through the shared provider router.
- Added routed data request/response types for non-search/fetch capabilities.
- Added quota preflight support with OpenAlex as the first wired endpoint.
- Added local provider usage counters and budget-based provider skipping.
- Stabilized evidence packs with schema version, normalized records, provenance fields, and content hashes.
- Expanded the public Go package with routed capability, quota, doctor, and evidence helpers.
- Removed the raw remote helper path from CLI command execution.
- Updated provider status and CLI documentation for quota preflight and metadata-only providers.

## v0.1.1

- Promoted Currents, Mediastack, World News API, SerpApi, and serpstack after adapter/smoke verification.
- Added a Semantic Scholar adapter but kept it metadata-only because unauthenticated local smoke checks returned 429.
- Added provider quota tracking metadata and persisted observed limit/remaining/used fields.
- Config repair for stale provider routes and removed provider cleanup.
- Mocked adapter coverage for live web, news, scholar, and fetch providers.
- CLI tests for config, missing config, and provider registry JSON output.
- Hardened quota header parsing, negative cache reads, auth-missing errors, URL normalization, snippet trimming, and extraction quality fallback.
- GDELT is metadata-only because local Windows smoke checks showed the public endpoint is unreliable from this environment.
- Provider verification status documented in `PROVIDER_STATUS.md`.

## v0.1.0

- Initial Windows-focused production baseline target.
- Repo-local `.forage/config.yaml`.
- OS keychain credentials with environment fallback.
- Provider registry, setup, doctor, quota, cache, search, fetch, extraction, and evidence-pack command surfaces.
- GitHub Releases workflow for Windows artifacts.
