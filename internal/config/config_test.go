package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitCreatesGlobalConfigWithoutSecrets(t *testing.T) {
	t.Chdir(t.TempDir())
	path, created, err := Init(false)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if !created {
		t.Fatal("Init() did not create config")
	}
	if path != filepath.Join(".forage", "config.yaml") {
		t.Fatalf("path = %q", path)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) == "" {
		t.Fatal("config is empty")
	}
	if Exists() != true {
		t.Fatal("Exists() = false")
	}
}

func TestLoadAppliesDefaults(t *testing.T) {
	t.Chdir(t.TempDir())
	if _, _, err := Init(false); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Profile != "free" {
		t.Fatalf("Profile = %q", cfg.Profile)
	}
	if cfg.Credentials.Store != "keychain" {
		t.Fatalf("Credentials.Store = %q", cfg.Credentials.Store)
	}
	if Enabled(cfg, "brave") {
		t.Fatal("brave should not be enabled by default")
	}
	if !Enabled(cfg, "direct") {
		t.Fatal("direct should be enabled by default")
	}
	if !Enabled(cfg, "browserbase") {
		t.Fatal("browserbase should be enabled by default")
	}
	for _, removed := range []string{"blogger", "wordpress", "wordpress_com", "diffbot", "google_cse", "reddit", "brave", "tavily", "exa", "serpapi", "serpstack", "guardian", "currents", "newsapi", "gnews", "mediastack", "worldnews", "hackernews", "forem", "apify", "browserless", "orcid", "wikidata", "gdelt"} {
		if Enabled(cfg, removed) {
			t.Fatalf("%s should not be enabled by default", removed)
		}
	}
	if _, ok := cfg.Routing["search.web"]; ok {
		t.Fatal("generic web search route should not be present")
	}
	if _, ok := cfg.Routing["search.news"]; ok {
		t.Fatal("news search route should not be present")
	}
}

func TestLoadMergesNewDefaultProviders(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.MkdirAll(".forage", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(".forage", "config.yaml"), []byte("version: 1\nproviders:\n  brave:\n    enabled: true\n  blogger:\n    enabled: true\nrouting:\n  search.web:\n    - brave\n    - jina\n  extract.article:\n    - jina\ncache:\n  database: .forage/state.db\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !Enabled(cfg, "direct") {
		t.Fatal("direct should be merged into older configs")
	}
	if len(cfg.Routing["fetch.url"]) == 0 {
		t.Fatal("default fetch route should be merged")
	}
	if Enabled(cfg, "blogger") || Enabled(cfg, "brave") {
		t.Fatal("removed provider should be pruned from older configs")
	}
	if _, ok := cfg.Routing["search.web"]; ok {
		t.Fatal("removed generic search route should be pruned")
	}
	if !contains(cfg.Routing["extract.article"], "browserbase") {
		t.Fatal("new default extract provider should be appended to older routes")
	}
}

func TestRepairWritesNormalizedConfig(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.MkdirAll(".forage", 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(".forage", "config.yaml")
	if err := os.WriteFile(path, []byte("version: 1\nproviders:\n  blogger:\n    enabled: true\n  brave:\n    enabled: true\nrouting:\n  search.web:\n    - google_cse\n  extract.article:\n    - apify\ncache:\n  database: .forage/state.db\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, changed, err := Repair()
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("Repair() changed = false")
	}
	if Enabled(cfg, "blogger") {
		t.Fatal("removed provider should be pruned")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "blogger") || strings.Contains(string(b), "google_cse") || strings.Contains(string(b), "brave") || strings.Contains(string(b), "apify") {
		t.Fatalf("repaired config still contains removed provider:\n%s", string(b))
	}
}
