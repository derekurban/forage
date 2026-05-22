# Changelog

## v0.1.1

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
