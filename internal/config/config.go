package config

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
)

const (
	DirName  = ".forage"
	FileName = "config.yaml"
	DBName   = "state.db"
)

type Config struct {
	Version     int                       `yaml:"version" json:"version"`
	Profile     string                    `yaml:"profile" json:"profile"`
	Policy      string                    `yaml:"policy" json:"policy"`
	MaxRequests int                       `yaml:"max_requests" json:"max_requests"`
	MaxFanout   int                       `yaml:"max_fanout" json:"max_fanout"`
	Budget      BudgetConfig              `yaml:"budget" json:"budget"`
	Credentials CredentialConfig          `yaml:"credentials" json:"credentials"`
	Providers   map[string]ProviderConfig `yaml:"providers" json:"providers"`
	Routing     map[string][]string       `yaml:"routing" json:"routing"`
	Cache       CacheConfig               `yaml:"cache" json:"cache"`
}

type CredentialConfig struct {
	Store string `yaml:"store" json:"store"`
}

type ProviderConfig struct {
	Enabled bool `yaml:"enabled" json:"enabled"`
}

type CacheConfig struct {
	Database string `yaml:"database" json:"database"`
	Mode     string `yaml:"mode" json:"mode"`
	TTLHours int    `yaml:"ttl_hours" json:"ttl_hours"`
}

type BudgetConfig struct {
	MaxCostUSD float64 `yaml:"max_cost_usd" json:"max_cost_usd"`
}

func Dir() string {
	return filepath.Join(".", DirName)
}

func Path() string {
	return filepath.Join(Dir(), FileName)
}

func DBPath() string {
	return filepath.Join(Dir(), DBName)
}

func Default() Config {
	return Config{
		Version:     1,
		Profile:     "free",
		Policy:      "research",
		MaxRequests: 50,
		MaxFanout:   4,
		Budget:      BudgetConfig{MaxCostUSD: 0},
		Credentials: CredentialConfig{Store: "keychain"},
		Providers: map[string]ProviderConfig{
			"crossref":         {Enabled: true},
			"arxiv":            {Enabled: true},
			"jina":             {Enabled: true},
			"direct":           {Enabled: true},
			"firecrawl":        {Enabled: true},
			"scrapingant":      {Enabled: true},
			"browserbase":      {Enabled: true},
			"openalex":         {Enabled: true},
			"semantic_scholar": {Enabled: true},
			"pubmed":           {Enabled: true},
			"datacite":         {Enabled: true},
			"opencitations":    {Enabled: true},
			"unpaywall":        {Enabled: true},
			"doaj":             {Enabled: true},
			"europepmc":        {Enabled: true},
			"internet_archive": {Enabled: true},
			"commoncrawl":      {Enabled: true},
		},
		Routing: map[string][]string{
			"search.scholar":  {"openalex", "crossref", "arxiv", "pubmed", "datacite", "europepmc", "doaj", "semantic_scholar"},
			"extract.article": {"jina", "browserbase", "firecrawl", "scrapingant", "direct"},
			"fetch.url":       {"jina", "browserbase", "firecrawl", "scrapingant", "direct"},
			"archive.lookup":  {"internet_archive", "commoncrawl"},
			"enrich.doi":      {"openalex", "unpaywall", "crossref"},
			"enrich.paper":    {"openalex", "semantic_scholar"},
			"citations.doi":   {"opencitations", "crossref", "openalex", "semantic_scholar"},
		},
		Cache: CacheConfig{Database: DBPath(), Mode: "auto", TTLHours: 24},
	}
}

func Exists() bool {
	_, err := os.Stat(Path())
	return err == nil
}

func Init(overwrite bool) (string, bool, error) {
	if err := os.MkdirAll(Dir(), 0o755); err != nil {
		return Path(), false, err
	}
	if Exists() && !overwrite {
		return Path(), false, nil
	}
	b, err := yaml.Marshal(Default())
	if err != nil {
		return Path(), false, err
	}
	return Path(), true, os.WriteFile(Path(), b, 0o600)
}

func Load() (Config, error) {
	b, err := os.ReadFile(Path())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, err
		}
		return Config{}, err
	}
	var cfg Config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return Config{}, err
	}
	if cfg.Version == 0 {
		cfg.Version = 1
	}
	if cfg.Profile == "" {
		cfg.Profile = "free"
	}
	if cfg.Policy == "" {
		cfg.Policy = "research"
	}
	if cfg.MaxRequests == 0 {
		cfg.MaxRequests = 50
	}
	if cfg.MaxFanout == 0 {
		cfg.MaxFanout = 4
	}
	if cfg.Credentials.Store == "" {
		cfg.Credentials.Store = "keychain"
	}
	if cfg.Cache.Database == "" {
		cfg.Cache.Database = DBPath()
	}
	if cfg.Cache.Mode == "" {
		cfg.Cache.Mode = "auto"
	}
	if cfg.Cache.TTLHours == 0 {
		cfg.Cache.TTLHours = 24
	}
	if cfg.Providers == nil {
		cfg.Providers = map[string]ProviderConfig{}
	}
	for id, pc := range Default().Providers {
		if _, ok := cfg.Providers[id]; !ok {
			cfg.Providers[id] = pc
		}
	}
	if cfg.Routing == nil {
		cfg.Routing = Default().Routing
	}
	for cap, route := range Default().Routing {
		if _, ok := cfg.Routing[cap]; !ok {
			cfg.Routing[cap] = route
		}
	}
	known := map[string]bool{}
	for id := range Default().Providers {
		known[id] = true
	}
	for id := range cfg.Providers {
		if !known[id] {
			delete(cfg.Providers, id)
		}
	}
	for cap, route := range cfg.Routing {
		defaultRoute, supportedCap := Default().Routing[cap]
		if !supportedCap {
			delete(cfg.Routing, cap)
			continue
		}
		var kept []string
		for _, id := range route {
			if known[id] {
				kept = append(kept, id)
			}
		}
		for _, id := range defaultRoute {
			if known[id] && !contains(kept, id) {
				kept = append(kept, id)
			}
		}
		cfg.Routing[cap] = kept
	}
	if routeEqual(cfg.Routing["enrich.doi"], []string{"crossref", "unpaywall", "openalex"}) {
		cfg.Routing["enrich.doi"] = Default().Routing["enrich.doi"]
	}
	return cfg, nil
}

func Repair() (Config, bool, error) {
	b, err := os.ReadFile(Path())
	if err != nil {
		return Config{}, false, err
	}
	cfg, err := Load()
	if err != nil {
		return Config{}, false, err
	}
	out, err := yaml.Marshal(cfg)
	if err != nil {
		return Config{}, false, err
	}
	changed := string(b) != string(out)
	if changed {
		if err := os.WriteFile(Path(), out, 0o600); err != nil {
			return Config{}, false, err
		}
	}
	return cfg, changed, nil
}

func Enabled(cfg Config, provider string) bool {
	p, ok := cfg.Providers[provider]
	if !ok {
		return false
	}
	return p.Enabled
}

func contains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func routeEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
