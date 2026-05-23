package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/derekurban/forage/internal/capability"
)

type rewriteTransport struct {
	base *url.URL
	rt   http.RoundTripper
}

func (t rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = t.base.Scheme
	req.URL.Host = t.base.Host
	return t.rt.RoundTrip(req)
}

func testClient(ts *httptest.Server) *http.Client {
	u, _ := url.Parse(ts.URL)
	return &http.Client{Transport: rewriteTransport{base: u, rt: http.DefaultTransport}}
}

func TestHTTPAdapterWebSearchProviders(t *testing.T) {
	tests := []struct {
		id      string
		method  string
		path    string
		body    string
		wantURL string
	}{
		{"brave", http.MethodGet, "/res/v1/web/search", `{"web":{"results":[{"title":"OpenAI","url":"https://openai.com/","description":"AI research"}]}}`, "https://openai.com/"},
		{"jina", http.MethodGet, "/openai", `{"data":[{"title":"OpenAI","url":"https://openai.com/","description":"AI research"}]}`, "https://openai.com/"},
		{"browserbase", http.MethodPost, "/v1/search", `{"results":[{"title":"OpenAI","url":"https://openai.com/","author":"OpenAI"}]}`, "https://openai.com/"},
		{"tavily", http.MethodPost, "/search", `{"results":[{"title":"OpenAI","url":"https://openai.com/","content":"AI research","score":0.9}]}`, "https://openai.com/"},
		{"exa", http.MethodPost, "/search", `{"results":[{"title":"OpenAI","url":"https://openai.com/","text":"AI research","score":0.9}]}`, "https://openai.com/"},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != tt.method {
					t.Fatalf("method = %s", r.Method)
				}
				if r.URL.Path != tt.path {
					t.Fatalf("path = %s", r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.body))
			}))
			defer ts.Close()
			a := HTTPAdapter{id: tt.id, client: testClient(ts), creds: fakeCreds{}}
			got, err := a.Search(context.Background(), capability.SearchRequest{Query: "openai", Capability: capability.SearchWeb, Limit: 1})
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != 1 || got[0].URL != tt.wantURL || got[0].Provider != tt.id {
				t.Fatalf("results = %+v", got)
			}
		})
	}
}

func TestHTTPAdapterNewsProviders(t *testing.T) {
	tests := []struct {
		id   string
		body string
	}{
		{"gdelt", `{"articles":[{"title":"Climate","url":"https://news.example/a","domain":"news.example"}]}`},
		{"brave", `{"results":[{"title":"Climate","url":"https://news.example/a","description":"news"}]}`},
		{"guardian", `{"response":{"results":[{"webTitle":"Climate","webUrl":"https://news.example/a","fields":{"trailText":"news"}}]}}`},
		{"gnews", `{"articles":[{"title":"Climate","url":"https://news.example/a","description":"news"}]}`},
		{"newsapi", `{"articles":[{"title":"Climate","url":"https://news.example/a","description":"news"}]}`},
		{"currents", `{"news":[{"title":"Climate","url":"https://news.example/a","description":"news"}]}`},
		{"mediastack", `{"data":[{"title":"Climate","url":"https://news.example/a","description":"news"}]}`},
		{"worldnews", `{"news":[{"title":"Climate","url":"https://news.example/a","text":"news"}]}`},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.body))
			}))
			defer ts.Close()
			a := HTTPAdapter{id: tt.id, client: testClient(ts), creds: fakeCreds{}}
			got, err := a.Search(context.Background(), capability.SearchRequest{Query: "climate", Capability: capability.SearchNews, Limit: 1})
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != 1 || got[0].ResultType != "news" {
				t.Fatalf("results = %+v", got)
			}
		})
	}
}

func TestHTTPAdapterScholarProviders(t *testing.T) {
	tests := []struct {
		id   string
		body string
	}{
		{"openalex", `{"results":[{"id":"W1","doi":"10.1/x","title":"Paper","publication_year":2024,"primary_location":{"landing_page_url":"https://example.com/p"}}]}`},
		{"crossref", `{"message":{"items":[{"DOI":"10.1/x","title":["Paper"],"URL":"https://example.com/p"}]}}`},
		{"pubmed", `{"esearchresult":{"idlist":["123"]}}|{"result":{"uids":["123"],"123":{"uid":"123","title":"Paper"}}}`},
		{"semantic_scholar", `{"data":[{"paperId":"S1","title":"Paper","abstract":"Abstract","url":"https://example.com/p","year":2024,"externalIds":{"DOI":"10.1/x"}}]}`},
		{"datacite", `{"data":[{"id":"10.1/x","attributes":{"doi":"10.1/x","titles":[{"title":"Paper"}],"url":"https://example.com/p","publicationYear":2024}}]}`},
		{"europepmc", `{"resultList":{"result":[{"id":"123","doi":"10.1/x","title":"Paper","abstractText":"Abstract"}]}}`},
		{"doaj", `{"results":[{"bibjson":{"title":"Paper","identifier":[{"type":"doi","id":"10.1/x"}],"link":[{"url":"https://example.com/p"}]}}]}`},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			parts := strings.Split(tt.body, "|")
			count := 0
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				body := parts[0]
				if count > 0 && len(parts) > 1 {
					body = parts[1]
				}
				count++
				_, _ = w.Write([]byte(body))
			}))
			defer ts.Close()
			a := HTTPAdapter{id: tt.id, client: testClient(ts), creds: fakeCreds{}}
			got, err := a.searchScholar(context.Background(), capability.SearchRequest{Query: "paper", Capability: capability.SearchScholar, Limit: 1})
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != 1 || got[0].Title == "" {
				t.Fatalf("works = %+v", got)
			}
		})
	}
}

func TestHTTPAdapterFetchProviders(t *testing.T) {
	tests := []struct {
		id   string
		body string
	}{
		{"jina", "Title\n\nThis is a complete article body long enough to pass quality checks in router tests."},
		{"browserbase", `{"statusCode":200,"content":"Title\n\nThis is a complete article body long enough to pass quality checks.","contentType":"text/markdown"}`},
		{"firecrawl", `{"success":true,"data":{"markdown":"Title\n\nThis is a complete article body long enough to pass quality checks.","html":"<h1>Title</h1>"}}`},
		{"scrapingant", "<html><head><title>Title</title></head><body><h1>Title</h1><p>This is a complete article body long enough to pass quality checks.</p></body></html>"},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.id == "browserbase" || tt.id == "firecrawl" {
					w.Header().Set("Content-Type", "application/json")
				}
				_, _ = w.Write([]byte(tt.body))
			}))
			defer ts.Close()
			a := HTTPAdapter{id: tt.id, client: testClient(ts), creds: fakeCreds{}}
			got, err := a.Fetch(context.Background(), capability.FetchRequest{URL: "https://example.com"})
			if err != nil {
				t.Fatal(err)
			}
			if got.Provider != tt.id || got.ContentHash == "" {
				t.Fatalf("doc = %+v", got)
			}
			if tt.id == "scrapingant" && strings.Contains(got.PlainText, "<html") {
				t.Fatalf("scrapingant plain text was not normalized: %+v", got)
			}
		})
	}
}

func TestHTTPAdapterArchiveEnrichAndCorpusProviders(t *testing.T) {
	tests := []struct {
		name string
		id   string
		run  func(HTTPAdapter) error
		body string
	}{
		{name: "internet_archive", id: "internet_archive", body: `{"archived_snapshots":{"closest":{"available":true,"url":"https://web.archive.org/x","timestamp":"20240101000000","status":"200"}}}`, run: func(a HTTPAdapter) error {
			got, err := a.LookupArchive(context.Background(), "https://example.com", 1)
			if err != nil {
				return err
			}
			if len(got) != 1 || got[0].Provider != "internet_archive" {
				t.Fatalf("archive = %+v", got)
			}
			return nil
		}},
		{name: "crossref_enrich", id: "crossref", body: `{"message":{"DOI":"10.1/x","title":["Paper"]}}`, run: func(a HTTPAdapter) error {
			_, err := a.Enrich(context.Background(), capability.DataRequest{Capability: capability.EnrichDOI, ID: "10.1/x"})
			return err
		}},
		{name: "unpaywall_enrich", id: "unpaywall", body: `{"doi":"10.1/x","is_oa":true,"oa_status":"gold","best_oa_location":{"url":"https://example.com/p","url_for_pdf":"https://example.com/p.pdf","license":"cc-by","host_type":"publisher"},"journal_name":"Journal","publisher":"Publisher"}`, run: func(a HTTPAdapter) error {
			got, err := a.Enrich(context.Background(), capability.DataRequest{Capability: capability.EnrichDOI, ID: "10.1/x"})
			if err != nil {
				return err
			}
			e, ok := got.(capability.ScholarlyEnrichment)
			if !ok || e.OpenAccess == nil || !e.OpenAccess.IsOA || e.OpenAccess.Raw != nil {
				t.Fatalf("unpaywall enrichment = %#v", got)
			}
			return nil
		}},
		{name: "semantic_scholar_enrich_doi", id: "semantic_scholar", body: `{"paperId":"S1","title":"Paper","abstract":"Abstract","url":"https://example.com/p","year":2024,"venue":"Venue","citationCount":7,"referenceCount":3,"externalIds":{"DOI":"10.1/x"},"authors":[{"name":"Ada Lovelace"}]}`, run: func(a HTTPAdapter) error {
			got, err := a.Enrich(context.Background(), capability.DataRequest{Capability: capability.EnrichPaper, ID: "10.1/x"})
			if err != nil {
				return err
			}
			e, ok := got.(capability.ScholarlyEnrichment)
			if !ok || e.Record == nil || e.Record.DOI != "10.1/x" || e.Record.CitationCount != 7 || len(e.Record.Authors) != 1 {
				t.Fatalf("semantic enrichment = %#v", got)
			}
			return nil
		}},
		{name: "commoncrawl_corpus", id: "commoncrawl", body: `[{"id":"CC-MAIN-2024-10","cdx-api":"https://index.example/","name":"Index"}]`, run: func(a HTTPAdapter) error {
			_, err := a.Corpus(context.Background(), capability.DataRequest{Capability: capability.CorpusQuery, Query: "x"})
			return err
		}},
		{name: "opencitations", id: "opencitations", body: `[{"citing":"10.2/y","cited":"10.1/x","creation":"2024-01-02"}]`, run: func(a HTTPAdapter) error {
			got, err := a.Citations(context.Background(), capability.DataRequest{Capability: capability.CitationsDOI, ID: "10.1/x"})
			if err != nil {
				return err
			}
			c, ok := got.(capability.CitationResponse)
			if !ok || len(c.Records) != 1 || c.Records[0].CitingDOI != "10.2/y" || c.Records[0].Year != 2024 {
				t.Fatalf("citations = %#v", got)
			}
			return nil
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.body))
			}))
			defer ts.Close()
			a := HTTPAdapter{id: tt.id, client: testClient(ts), creds: fakeCreds{}}
			if err := tt.run(a); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestSemanticScholarDOILookupUsesDOIPrefix(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.EscapedPath(), "DOI:10.1%2Fx") && !strings.Contains(r.URL.Path, "DOI:10.1/x") {
			t.Fatalf("path = %s escaped=%s", r.URL.Path, r.URL.EscapedPath())
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"paperId":"S1","title":"Paper","externalIds":{"DOI":"10.1/x"}}`))
	}))
	defer ts.Close()
	a := HTTPAdapter{id: "semantic_scholar", client: testClient(ts), creds: fakeCreds{}}
	got, err := a.Enrich(context.Background(), capability.DataRequest{Capability: capability.EnrichPaper, ID: "10.1/x"})
	if err != nil {
		t.Fatal(err)
	}
	e := got.(capability.ScholarlyEnrichment)
	if e.Record == nil || e.Record.DOI != "10.1/x" {
		t.Fatalf("enrichment = %#v", got)
	}
}

func TestRawPayloadsRequireIncludeRaw(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"doi":"10.1/x","is_oa":true,"best_oa_location":{"url":"https://example.com"}}`))
	}))
	defer ts.Close()
	a := HTTPAdapter{id: "unpaywall", client: testClient(ts), creds: fakeCreds{}}
	withoutRaw, err := a.Enrich(context.Background(), capability.DataRequest{Capability: capability.EnrichDOI, ID: "10.1/x"})
	if err != nil {
		t.Fatal(err)
	}
	if withoutRaw.(capability.ScholarlyEnrichment).Raw != nil || withoutRaw.(capability.ScholarlyEnrichment).OpenAccess.Raw != nil {
		t.Fatalf("raw leaked without flag: %#v", withoutRaw)
	}
	withRaw, err := a.Enrich(context.Background(), capability.DataRequest{Capability: capability.EnrichDOI, ID: "10.1/x", IncludeRaw: true})
	if err != nil {
		t.Fatal(err)
	}
	if withRaw.(capability.ScholarlyEnrichment).Raw == nil || withRaw.(capability.ScholarlyEnrichment).OpenAccess.Raw == nil {
		t.Fatalf("raw missing with flag: %#v", withRaw)
	}
}

func TestHTTPAdapterProviderFailures(t *testing.T) {
	tests := []struct {
		status int
		code   string
	}{
		{http.StatusUnauthorized, "auth_failed"},
		{http.StatusForbidden, "auth_failed"},
		{http.StatusTooManyRequests, "rate_limited"},
		{http.StatusInternalServerError, "provider_http_error"},
	}
	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Retry-After", "60")
				http.Error(w, "nope", tt.status)
			}))
			defer ts.Close()
			a := HTTPAdapter{id: "brave", client: testClient(ts), creds: fakeCreds{}}
			_, err := a.Search(context.Background(), capability.SearchRequest{Query: "x", Capability: capability.SearchWeb, Limit: 1})
			pe, ok := err.(ProviderError)
			if !ok {
				t.Fatalf("err = %#v", err)
			}
			if pe.Code != tt.code {
				t.Fatalf("code = %s", pe.Code)
			}
		})
	}
}

func TestOptionalAPIKeyRetriesWithoutKey(t *testing.T) {
	count := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("api_key") != "" {
			_, _ = w.Write([]byte(`<ERROR>bad key</ERROR>`))
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer ts.Close()
	a := HTTPAdapter{id: "pubmed", client: testClient(ts), creds: fakeCreds{}}
	var raw map[string]any
	if err := a.getJSONWithOptionalAPIKey(context.Background(), "https://eutils.ncbi.nlm.nih.gov/test", "api_key", "bad", &raw); err != nil {
		t.Fatal(err)
	}
	if count != 2 || raw["ok"] != true {
		t.Fatalf("count=%d raw=%+v", count, raw)
	}
}

func TestHTTPAdapterNormalizesCommaSeparatedRateLimitHeaders(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "5, 10")
		w.Header().Set("x-ratelimit-reset", "1, 788324")
		http.Error(w, "limited", http.StatusTooManyRequests)
	}))
	defer ts.Close()
	a := HTTPAdapter{id: "brave", client: testClient(ts), creds: fakeCreds{}}
	_, err := a.Search(context.Background(), capability.SearchRequest{Query: "x", Capability: capability.SearchWeb, Limit: 1})
	pe, ok := err.(ProviderError)
	if !ok {
		t.Fatalf("err = %#v", err)
	}
	if pe.RetryAfter != "5" || pe.ResetAt != "1" {
		t.Fatalf("retry/reset = %q/%q", pe.RetryAfter, pe.ResetAt)
	}
}

func TestParseObservedQuota(t *testing.T) {
	limit, remaining, used := parseObservedQuota("x-ratelimit-limit=100, 1000; x-ratelimit-remaining=42, 999; x-ratelimit-credits-used=3")
	if limit == nil || *limit != 100 {
		t.Fatalf("limit = %+v", limit)
	}
	if remaining == nil || *remaining != 42 {
		t.Fatalf("remaining = %+v", remaining)
	}
	if used == nil || *used != 3 {
		t.Fatalf("used = %+v", used)
	}
}
