package router

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/derekurban/forage/internal/capability"
	"github.com/derekurban/forage/internal/config"
	"github.com/derekurban/forage/internal/credentials"
	"github.com/derekurban/forage/internal/state"
)

type fakeCreds struct{}

func (fakeCreds) Get(provider, envVar string) (credentials.Credential, error) {
	return credentials.Credential{Found: true, Source: "test", Value: "x"}, nil
}
func (fakeCreds) Set(provider, value string) error { return nil }
func (fakeCreds) Delete(provider string) error     { return nil }
func (fakeCreds) GetField(provider, field, envVar string) (credentials.Credential, error) {
	return credentials.Credential{Found: true, Source: "test", Value: "x"}, nil
}
func (fakeCreds) SetField(provider, field, value string) error { return nil }
func (fakeCreds) DeleteField(provider, field string) error     { return nil }

type missingCreds struct{}

func (missingCreds) Get(provider, envVar string) (credentials.Credential, error) {
	return credentials.Credential{Found: false, Source: "missing"}, nil
}
func (missingCreds) Set(provider, value string) error { return nil }
func (missingCreds) Delete(provider string) error     { return nil }
func (missingCreds) GetField(provider, field, envVar string) (credentials.Credential, error) {
	return credentials.Credential{Found: false, Source: "missing"}, nil
}
func (missingCreds) SetField(provider, field, value string) error { return nil }
func (missingCreds) DeleteField(provider, field string) error     { return nil }

type fakeSearch struct {
	id  string
	err error
}

func (f fakeSearch) ID() string { return f.id }
func (f fakeSearch) Search(ctx context.Context, req capability.SearchRequest) ([]capability.SearchResult, error) {
	if f.err != nil {
		return nil, f.err
	}
	return []capability.SearchResult{{URL: "https://example.com/" + f.id, Title: f.id, Provider: f.id, ResultType: "web"}}, nil
}

type fakeFetch struct {
	id  string
	doc capability.ExtractedDocument
	err error
}

func (f fakeFetch) ID() string { return f.id }
func (f fakeFetch) Fetch(ctx context.Context, req capability.FetchRequest) (capability.ExtractedDocument, error) {
	if f.err != nil {
		return capability.ExtractedDocument{}, f.err
	}
	f.doc.Provider = f.id
	return f.doc, nil
}

func TestSearchFallsBackOnProviderError(t *testing.T) {
	st, err := state.Open(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	cfg := config.Default()
	cfg.Routing[capability.SearchWeb] = []string{"bad", "good"}
	cfg.Providers["bad"] = config.ProviderConfig{Enabled: true}
	cfg.Providers["good"] = config.ProviderConfig{Enabled: true}
	r := New(cfg, st, fakeCreds{})
	r.Adapters = map[string]Adapter{
		"bad":  fakeSearch{id: "bad", err: ProviderError{Code: "rate_limited", Message: "limited", HTTPStatus: 429}},
		"good": fakeSearch{id: "good"},
	}
	resp, ae := r.Search(context.Background(), capability.SearchRequest{Query: "x", Capability: capability.SearchWeb, Limit: 1, ExplainRouting: true, CacheMode: "refresh"})
	if ae != nil {
		t.Fatalf("Search error = %v", ae)
	}
	if len(resp.Results) != 1 || resp.Results[0].Provider != "good" {
		t.Fatalf("unexpected results: %+v", resp.Results)
	}
	if len(resp.Routing.Attempts) < 2 {
		t.Fatalf("missing attempts: %+v", resp.Routing)
	}
}

func TestSearchExhausted(t *testing.T) {
	st, err := state.Open(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	cfg := config.Default()
	cfg.Routing[capability.SearchWeb] = []string{"bad"}
	cfg.Providers["bad"] = config.ProviderConfig{Enabled: true}
	r := New(cfg, st, fakeCreds{})
	r.Adapters = map[string]Adapter{"bad": fakeSearch{id: "bad", err: errors.New("boom")}}
	_, ae := r.Search(context.Background(), capability.SearchRequest{Query: "x", Capability: capability.SearchWeb, Limit: 1, CacheMode: "refresh"})
	if ae == nil || ae.Code != "capability_exhausted" {
		t.Fatalf("expected exhausted, got %+v", ae)
	}
}

func TestSearchSkipsProviderInCooldown(t *testing.T) {
	st, err := state.Open(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	cfg := config.Default()
	cfg.Routing[capability.SearchWeb] = []string{"cool", "good"}
	cfg.Providers["cool"] = config.ProviderConfig{Enabled: true}
	cfg.Providers["good"] = config.ProviderConfig{Enabled: true}
	if err := st.UpsertProviderState(state.ProviderState{Provider: "cool", Status: "cooldown", Reason: "rate_limited", RetryAfter: "120"}); err != nil {
		t.Fatal(err)
	}
	r := New(cfg, st, fakeCreds{})
	r.Adapters = map[string]Adapter{
		"cool": fakeSearch{id: "cool"},
		"good": fakeSearch{id: "good"},
	}
	resp, ae := r.Search(context.Background(), capability.SearchRequest{Query: "x", Capability: capability.SearchWeb, Limit: 1, ExplainRouting: true, CacheMode: "refresh"})
	if ae != nil {
		t.Fatalf("Search error = %v", ae)
	}
	if resp.Results[0].Provider != "good" {
		t.Fatalf("provider = %s", resp.Results[0].Provider)
	}
	if len(resp.Routing.Skipped) == 0 || resp.Routing.Skipped[0].Reason != "cooldown" {
		t.Fatalf("expected cooldown skip, got %+v", resp.Routing.Skipped)
	}
}

func TestSearchAllowsExpiredCooldown(t *testing.T) {
	st, err := state.Open(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	cfg := config.Default()
	cfg.Routing[capability.SearchWeb] = []string{"cool"}
	cfg.Providers["cool"] = config.ProviderConfig{Enabled: true}
	checked := time.Now().UTC().Add(-5 * time.Minute).Format(time.RFC3339)
	if err := st.UpsertProviderState(state.ProviderState{Provider: "cool", Status: "cooldown", Reason: "rate_limited", RetryAfter: "60", LastCheckedAt: checked}); err != nil {
		t.Fatal(err)
	}
	r := New(cfg, st, fakeCreds{})
	r.Adapters = map[string]Adapter{"cool": fakeSearch{id: "cool"}}
	resp, ae := r.Search(context.Background(), capability.SearchRequest{Query: "x", Capability: capability.SearchWeb, Limit: 1, ExplainRouting: true, CacheMode: "refresh"})
	if ae != nil {
		t.Fatalf("Search error = %v", ae)
	}
	if resp.Results[0].Provider != "cool" {
		t.Fatalf("provider = %s", resp.Results[0].Provider)
	}
}

func TestSearchMissingAuthReturnsAuthMissing(t *testing.T) {
	st, err := state.Open(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	cfg := config.Default()
	cfg.Routing[capability.SearchWeb] = []string{"brave", "tavily"}
	r := New(cfg, st, missingCreds{})
	r.Adapters = map[string]Adapter{}
	_, ae := r.Search(context.Background(), capability.SearchRequest{Query: "x", Capability: capability.SearchWeb, Limit: 1, CacheMode: "refresh", ExplainRouting: true})
	if ae == nil || ae.Code != "auth_missing" {
		t.Fatalf("expected auth_missing, got %+v", ae)
	}
}

func TestSearchReadsNegativeCache(t *testing.T) {
	st, err := state.Open(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	cfg := config.Default()
	req := capability.SearchRequest{Query: "x", Capability: capability.SearchWeb, Limit: 1, CacheMode: "auto"}
	if err := st.PutNegativeCache(cacheKey("search", req), "provider_timeout", time.Minute); err != nil {
		t.Fatal(err)
	}
	r := New(cfg, st, fakeCreds{})
	_, ae := r.Search(context.Background(), req)
	if ae == nil || ae.Code != "cache_miss" {
		t.Fatalf("expected cache_miss, got %+v", ae)
	}
}

func TestFetchFallsBackOnPoorQuality(t *testing.T) {
	st, err := state.Open(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	cfg := config.Default()
	cfg.Routing[capability.FetchURL] = []string{"bad", "good"}
	cfg.Providers["bad"] = config.ProviderConfig{Enabled: true}
	cfg.Providers["good"] = config.ProviderConfig{Enabled: true}
	r := New(cfg, st, fakeCreds{})
	r.Adapters = map[string]Adapter{
		"bad":  fakeFetch{id: "bad", doc: capability.ExtractedDocument{URL: "https://example.com", Markdown: "captcha", QualityScore: 0.1}},
		"good": fakeFetch{id: "good", doc: capability.ExtractedDocument{URL: "https://example.com", Markdown: "This is a complete extracted document with enough body text to pass the quality threshold for routing.", QualityScore: 0.8}},
	}
	resp, ae := r.Fetch(context.Background(), capability.FetchRequest{URL: "https://example.com", CacheMode: "refresh", ExplainRouting: true})
	if ae != nil {
		t.Fatalf("Fetch error = %v", ae)
	}
	if resp.Document.Provider != "good" {
		t.Fatalf("provider = %s", resp.Document.Provider)
	}
	if resp.Routing.Attempts[0].Reason != "poor_quality" {
		t.Fatalf("expected poor_quality, got %+v", resp.Routing.Attempts)
	}
}

func TestResultNormalizesAndTrimsSnippet(t *testing.T) {
	long := ""
	for i := 0; i < 1000; i++ {
		long += "x"
	}
	got := result("test", 0, "HTTPS://Example.com/path?utm_source=x&a=b#frag", " title ", long, "web")
	if got.CanonicalURL != "https://example.com/path?a=b" {
		t.Fatalf("canonical_url = %q", got.CanonicalURL)
	}
	if len(got.Snippet) > 703 {
		t.Fatalf("snippet too long: %d", len(got.Snippet))
	}
}

func TestStripHTMLRemovesStyleAndScript(t *testing.T) {
	got := stripHTML(`<html><head><style>body{color:red}</style><script>alert(1)</script></head><body><h1>Title</h1><p>A &amp; B</p></body></html>`)
	if got != "Title A & B" {
		t.Fatalf("stripHTML() = %q", got)
	}
}
