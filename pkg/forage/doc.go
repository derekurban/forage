// Package forage exposes the stable library surface behind the forage CLI.
//
// The package loads the repo-local .forage/config.yaml, uses the same
// credential lookup and SQLite state as the CLI, and keeps provider-specific
// adapters internal. The public surface intentionally mirrors the complementary
// primitives: extraction, scholarly lookup/enrichment, archive lookup, doctor,
// quota, and evidence pack creation.
package forage
