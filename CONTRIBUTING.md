# Contributing

Forage is currently Windows-first. Please keep changes tested on Windows and avoid CGO-only dependencies.

## Checks

```powershell
go test ./...
go build ./cmd/forage
```

Provider adapters should use `httptest.Server` coverage for auth failures, quota responses, malformed payloads, empty results, and fallback behavior. Live tests must be opt-in with `FORAGE_LIVE_TESTS=1`.
