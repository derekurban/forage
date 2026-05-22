package router

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

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

func TestStripHTMLRemovesStyleAndScript(t *testing.T) {
	got := stripHTML(`<html><head><style>body{color:red}</style><script>alert(1)</script></head><body><h1>Title</h1><p>A &amp; B</p></body></html>`)
	if got != "Title A & B" {
		t.Fatalf("stripHTML() = %q", got)
	}
}
