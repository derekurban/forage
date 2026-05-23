package doctor

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/derekurban/forage/internal/config"
	"github.com/derekurban/forage/internal/credentials"
	"github.com/derekurban/forage/internal/providers"
	"github.com/derekurban/forage/internal/state"
)

type Result struct {
	Provider         providers.Provider  `json:"provider"`
	ConfigStatus     string              `json:"config_status"`
	HealthStatus     string              `json:"health_status"`
	CredentialSource string              `json:"credential_source,omitempty"`
	Message          string              `json:"message"`
	State            state.ProviderState `json:"state"`
}

type Runner struct {
	Client *http.Client
	Creds  credentials.Store
	State  *state.Store
}

func New(creds credentials.Store, st *state.Store) Runner {
	return Runner{
		Client: &http.Client{Timeout: 10 * time.Second},
		Creds:  creds,
		State:  st,
	}
}

func (r Runner) Check(ctx context.Context, cfg config.Config, p providers.Provider) Result {
	res := Result{Provider: p}
	if !config.Enabled(cfg, p.ID) {
		res.ConfigStatus = "disabled"
		res.HealthStatus = "disabled"
		res.Message = "provider disabled in config"
		res.State = newState(p.ID, "disabled", "disabled_in_config")
		_ = r.State.UpsertProviderState(res.State)
		return res
	}
	res.ConfigStatus = "enabled"
	if p.Status == providers.LegacyOptional {
		res.HealthStatus = "legacy_optional"
		res.Message = "legacy optional provider; not included in default setup or routing"
		res.State = newState(p.ID, "disabled", "legacy_optional")
		_ = r.State.UpsertProviderState(res.State)
		return res
	}
	if p.Status == providers.MetadataOnly {
		res.HealthStatus = "metadata_only"
		res.Message = "provider metadata registered; live doctor check not implemented yet"
		res.State = newState(p.ID, "degraded", "metadata_only")
		_ = r.State.UpsertProviderState(res.State)
		return res
	}
	var key string
	if len(p.CredentialFields) > 0 {
		var missing []string
		var sources []string
		for _, f := range p.CredentialFields {
			cred, err := r.Creds.GetField(p.ID, f.Name, f.EnvVar)
			if err != nil || !cred.Found {
				if f.Required {
					missing = append(missing, f.Name)
				}
				continue
			}
			if f.Name == "api_key" || key == "" {
				key = cred.Value
			}
			sources = append(sources, f.Name+"="+cred.Source)
		}
		if len(missing) > 0 {
			res.HealthStatus = "missing"
			res.Message = fmt.Sprintf("missing credentials %s; run `forage setup` or `forage credentials set %s --field FIELD --value-stdin`", strings.Join(missing, ","), p.ID)
			res.State = newState(p.ID, "disabled", "auth_missing")
			_ = r.State.UpsertProviderState(res.State)
			return res
		}
		if len(sources) > 0 {
			res.CredentialSource = strings.Join(sources, "; ")
		} else if p.OptionalAuth {
			res.CredentialSource = "none_optional"
		}
	}
	res.HealthStatus, res.Message, res.State = r.probe(ctx, p, key)
	_ = r.State.UpsertProviderState(res.State)
	return res
}

func newState(provider, status, reason string) state.ProviderState {
	return state.ProviderState{
		Provider:      provider,
		Status:        status,
		Reason:        reason,
		LastCheckedAt: time.Now().UTC().Format(time.RFC3339),
	}
}

func (r Runner) probe(ctx context.Context, p providers.Provider, key string) (string, string, state.ProviderState) {
	now := time.Now().UTC().Format(time.RFC3339)
	ps := state.ProviderState{Provider: p.ID, Status: "healthy", LastCheckedAt: now}
	var req *http.Request
	var err error
	switch p.ID {
	case "direct":
		ps.Status = "healthy"
		ps.LastSuccessAt = now
		return "healthy", "local direct fetch is available", ps
	case "hackernews":
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, "https://hacker-news.firebaseio.com/v0/topstories.json", nil)
	case "crossref":
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, "https://api.crossref.org/works?query=forage&rows=1", nil)
		req.Header.Set("User-Agent", "forage/0.1 (mailto:unknown@example.com)")
	case "arxiv":
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, "https://export.arxiv.org/api/query?search_query=all:forage&start=0&max_results=1", nil)
	case "brave":
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, "https://api.search.brave.com/res/v1/web/search?q=forage&count=1", nil)
		req.Header.Set("X-Subscription-Token", key)
	case "jina":
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, "https://r.jina.ai/https://www.example.com", nil)
		if key != "" {
			req.Header.Set("Authorization", "Bearer "+key)
		}
	case "tavily":
		req, err = http.NewRequestWithContext(ctx, http.MethodPost, "https://api.tavily.com/search", strings.NewReader(`{"query":"forage","max_results":1}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+key)
	case "exa":
		req, err = http.NewRequestWithContext(ctx, http.MethodPost, "https://api.exa.ai/search", strings.NewReader(`{"query":"forage","numResults":1}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", key)
	case "browserbase":
		req, err = http.NewRequestWithContext(ctx, http.MethodPost, "https://api.browserbase.com/v1/search", strings.NewReader(`{"query":"forage","numResults":1}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-BB-API-Key", key)
	case "firecrawl":
		req, err = http.NewRequestWithContext(ctx, http.MethodPost, "https://api.firecrawl.dev/v1/scrape", strings.NewReader(`{"url":"https://www.example.com","formats":["markdown"]}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+key)
	case "scrapingant":
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, "https://api.scrapingant.com/v2/general?url=https%3A%2F%2Fwww.example.com&return_text=true&x-api-key="+key, nil)
	case "guardian":
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, "https://content.guardianapis.com/search?q=forage&page-size=1&api-key="+key, nil)
	case "gnews":
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, "https://gnews.io/api/v4/search?q=forage&max=1&apikey="+key, nil)
	case "newsapi":
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, "https://newsapi.org/v2/everything?q=forage&pageSize=1&apiKey="+key, nil)
	case "openalex":
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, "https://api.openalex.org/works?search=forage&per-page=1", nil)
		if key != "" {
			q := req.URL.Query()
			q.Set("api_key", key)
			req.URL.RawQuery = q.Encode()
		}
	case "semantic_scholar":
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, "https://api.semanticscholar.org/graph/v1/paper/search?query=forage&limit=1&fields=title", nil)
	case "opencitations":
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, "https://opencitations.net/index/api/v1/citations/10.1038/nature12373", nil)
		if key != "" {
			req.Header.Set("authorization", key)
		}
	case "unpaywall":
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, "https://api.unpaywall.org/v2/10.1038/nature12373?email="+url.QueryEscape(key), nil)
	case "pubmed":
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, "https://eutils.ncbi.nlm.nih.gov/entrez/eutils/esearch.fcgi?db=pubmed&retmode=json&term=forage&retmax=1", nil)
		if key != "" {
			q := req.URL.Query()
			q.Set("api_key", key)
			req.URL.RawQuery = q.Encode()
		}
	case "datacite":
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, "https://api.datacite.org/dois?query=forage&page[size]=1", nil)
	case "europepmc":
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, "https://www.ebi.ac.uk/europepmc/webservices/rest/search?format=json&query=forage&pageSize=1", nil)
	case "doaj":
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, "https://doaj.org/api/search/articles/forage?pageSize=1", nil)
	case "forem":
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, "https://dev.to/api/articles?tag=go&per_page=1", nil)
	case "gdelt":
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, "https://api.gdeltproject.org/api/v2/doc/doc?query=forage&mode=artlist&format=json&maxrecords=1", nil)
	case "internet_archive":
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, "https://archive.org/wayback/available?url=example.com", nil)
	case "commoncrawl":
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, "https://index.commoncrawl.org/collinfo.json", nil)
	default:
		ps.Status = "degraded"
		ps.Reason = "probe_not_implemented"
		return "metadata_only", "live doctor check not implemented", ps
	}
	if err != nil {
		ps.Status = "degraded"
		ps.Reason = "request_build_failed"
		return "error", err.Error(), ps
	}
	resp, err := r.Client.Do(req)
	if err != nil {
		ps.Status = "degraded"
		ps.Reason = "request_failed"
		ps.Observed = err.Error()
		return "error", err.Error(), ps
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	ps.LastHTTPStatus = &resp.StatusCode
	observeHeaders(resp, &ps)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		ps.Status = "healthy"
		ps.LastSuccessAt = now
		return "healthy", "health check succeeded", ps
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		ps.Status = "cooldown"
		ps.Reason = "rate_limited"
		return "cooldown", "provider is rate limited", ps
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		ps.Status = "disabled"
		ps.Reason = "auth_failed"
		return "missing", "credential rejected by provider", ps
	}
	ps.Status = "degraded"
	ps.Reason = "unexpected_status"
	return "error", fmt.Sprintf("provider returned HTTP %d", resp.StatusCode), ps
}

func observeHeaders(resp *http.Response, ps *state.ProviderState) {
	for _, h := range []string{"x-ratelimit-remaining", "ratelimit-remaining", "x-rate-limit-remaining"} {
		if v := resp.Header.Get(h); v != "" {
			if n, err := strconv.ParseInt(firstHeaderValue(v), 10, 64); err == nil {
				ps.Remaining = &n
			}
			break
		}
	}
	for _, h := range []string{"x-ratelimit-limit", "ratelimit-limit", "x-rate-limit-limit"} {
		if v := resp.Header.Get(h); v != "" {
			if n, err := strconv.ParseInt(firstHeaderValue(v), 10, 64); err == nil {
				ps.Limit = &n
			}
			break
		}
	}
	for _, h := range []string{"x-ratelimit-used", "ratelimit-used", "x-ratelimit-credits-used"} {
		if v := resp.Header.Get(h); v != "" {
			if n, err := strconv.ParseInt(firstHeaderValue(v), 10, 64); err == nil {
				ps.Used = &n
			}
			break
		}
	}
	if v := resp.Header.Get("retry-after"); v != "" {
		ps.RetryAfter = v
	}
	for _, h := range []string{"x-ratelimit-reset", "ratelimit-reset", "x-rate-limit-reset"} {
		if v := resp.Header.Get(h); v != "" {
			ps.ResetAt = firstHeaderValue(v)
			break
		}
	}
	var observed []string
	for _, h := range []string{"x-ratelimit-limit", "x-ratelimit-remaining", "x-ratelimit-reset", "ratelimit-limit", "ratelimit-remaining", "ratelimit-reset", "retry-after"} {
		if v := resp.Header.Get(h); v != "" {
			observed = append(observed, h+"="+v)
		}
	}
	ps.Observed = strings.Join(observed, "; ")
}

func firstHeaderValue(v string) string {
	parts := strings.Split(v, ",")
	if len(parts) == 0 {
		return strings.TrimSpace(v)
	}
	return strings.TrimSpace(parts[0])
}
