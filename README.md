# Forage

Forage is a Windows-first Go CLI for provider-aware research retrieval. It gives agents and shell users stable commands for search, fetch, extraction, provider setup, health checks, quota state, caching, and evidence packs while hiding provider-specific fallback and rate-limit handling during successful runs.

Forage is not a research agent or report writer. It is the retrieval and evidence I/O layer underneath one.

## Status

`v0.1.0` is the production baseline target. The CLI includes a broad recurring-free provider registry, live support for the first provider set, credential and doctor workflows, SQLite state, and stable JSON output. Providers that are not verified live are exposed honestly as credential-only, metadata-only, or legacy optional.

## Install on Windows

Download the latest Windows release zip from GitHub Releases, expand it, and put `forage.exe` on your `PATH`.

PowerShell:

```powershell
forage version
forage config init
forage providers list
```

From source:

```powershell
go install github.com/derekurban/forage/cmd/forage@latest
```

## Configuration

Forage uses one repo-local global config:

```text
.forage/config.yaml
```

There is no user-home config, project precedence, or scoped permission model in `v0.1.0`.

Create it with:

```powershell
forage config init
```

Secrets are stored in the OS keychain by default. Environment variables are supported as fallback/override, but their values are never persisted to config.

## Common Commands

```powershell
forage setup
forage credentials list
forage providers doctor
forage providers quota
forage search web "query" --json
forage search scholar "machine learning" --json
forage fetch https://example.com --json
forage extract urls.txt --jsonl
forage research-pack "machine learning" --json
```

Use `--explain-routing` or `--verbose` to inspect provider attempts. Normal output hides intermediate provider failures when fallback succeeds.

## Provider Policy

The default registry includes providers that appear to support recurring free usage, free public APIs, or free-account recurring quotas. Trial-only, paid-only, and one-off credit offers are not included in default setup or routing.

If no configured provider can satisfy a command, Forage fails clearly with a structured error and setup hints instead of inventing weak fallback sources.

## Development

```powershell
go test ./...
go build ./cmd/forage
```

Optional live tests should be gated behind `FORAGE_LIVE_TESTS=1` and provider-specific environment variables to avoid accidental quota usage.
