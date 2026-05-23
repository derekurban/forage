package agent

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/derekurban/forage/internal/capability"
	"github.com/derekurban/forage/internal/config"
	"github.com/derekurban/forage/internal/credentials"
	"github.com/derekurban/forage/internal/router"
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

type fakeSearch struct{}

func (fakeSearch) ID() string { return "searcher" }
func (fakeSearch) Search(ctx context.Context, req capability.SearchRequest) ([]capability.SearchResult, error) {
	return []capability.SearchResult{
		{URL: "https://example.com/a", CanonicalURL: "https://example.com/a", Title: "A", Snippet: "snippet A", Provider: "searcher", SourceDomain: "example.com", ResultType: "web"},
		{URL: "https://example.com/a", CanonicalURL: "https://example.com/a", Title: "A duplicate", Snippet: "snippet duplicate", Provider: "searcher", SourceDomain: "example.com", ResultType: "web"},
		{URL: "https://example.com/b", CanonicalURL: "https://example.com/b", Title: "B", Snippet: "snippet B", Provider: "searcher", SourceDomain: "example.com", ResultType: "web"},
	}, nil
}

type fakeFetch struct{}

func (fakeFetch) ID() string { return "fetcher" }
func (fakeFetch) Fetch(ctx context.Context, req capability.FetchRequest) (capability.ExtractedDocument, error) {
	return capability.ExtractedDocument{URL: req.URL, Title: "Fetched", RetrievedAt: time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC), Markdown: strings.Repeat("body ", 40), Provider: "fetcher", ExtractionMethod: "test", ContentHash: "hash", QualityScore: .9}, nil
}

func testWorkflow(t *testing.T) Workflow {
	t.Helper()
	st, err := state.Open(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	cfg := config.Default()
	cfg.Routing[capability.SearchWeb] = []string{"searcher"}
	cfg.Routing[capability.FetchURL] = []string{"fetcher"}
	cfg.Providers["searcher"] = config.ProviderConfig{Enabled: true}
	cfg.Providers["fetcher"] = config.ProviderConfig{Enabled: true}
	r := router.New(cfg, st, fakeCreds{})
	r.Adapters = map[string]router.Adapter{"searcher": fakeSearch{}, "fetcher": fakeFetch{}}
	return Workflow{Router: r, Config: cfg, EvidenceDir: t.TempDir()}
}

func TestClassifyInput(t *testing.T) {
	tests := map[string]string{
		"https://example.com":     KindURL,
		"https://doi.org/10.1/x":  KindDOI,
		"10.1038/nature12373":     KindDOI,
		"0000-0002-1825-0097":     KindAuthor,
		"2401.12345":              KindPaper,
		"42172029":                KindPaper,
		"latest browserbase docs": KindQuery,
	}
	for input, want := range tests {
		if got := ClassifyInput(input); got != want {
			t.Fatalf("ClassifyInput(%q) = %s, want %s", input, got, want)
		}
	}
}

func TestGatherDedupesAndFetchesLimit(t *testing.T) {
	wf := testWorkflow(t)
	resp, ae := wf.Gather(context.Background(), GatherRequest{Query: "x", Limit: 3, Fetch: 1, CacheMode: "refresh", ExplainRouting: true})
	if ae != nil {
		t.Fatal(ae)
	}
	if len(resp.Records) != 1 {
		t.Fatalf("records = %+v", resp.Records)
	}
	if resp.Records[0].SearchProvider != "searcher" || resp.Records[0].FetchProvider != "fetcher" {
		t.Fatalf("record provenance = %+v", resp.Records[0])
	}
	if resp.Routing == nil || len(resp.Routing.Search) != 1 || len(resp.Routing.Fetches) != 1 {
		t.Fatalf("routing = %+v", resp.Routing)
	}
}

func TestBriefRendersContext(t *testing.T) {
	wf := testWorkflow(t)
	resp, ae := wf.Brief(context.Background(), BriefRequest{Query: "x", Limit: 2, Fetch: 1, MaxChars: 25, CacheMode: "refresh"})
	if ae != nil {
		t.Fatal(ae)
	}
	if !strings.Contains(resp.Context, "SOURCE 1") || !strings.Contains(resp.Context, "Providers: searcher fetcher") {
		t.Fatalf("context = %s", resp.Context)
	}
	if len(resp.Records[0].Text) > 28 {
		t.Fatalf("text was not trimmed: %q", resp.Records[0].Text)
	}
}

func TestRetrieveURLAndSavePack(t *testing.T) {
	wf := testWorkflow(t)
	resp, ae := wf.Retrieve(context.Background(), RetrieveRequest{Input: "https://example.com/a", Kind: KindAuto, CacheMode: "refresh", SavePack: true})
	if ae != nil {
		t.Fatal(ae)
	}
	if resp.Kind != KindURL || len(resp.Records) != 1 || resp.EvidencePackPath == "" {
		t.Fatalf("retrieve = %+v", resp)
	}
}
