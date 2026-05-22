package router

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/derekurban/forage/internal/apperr"
	"github.com/derekurban/forage/internal/capability"
	"github.com/derekurban/forage/internal/config"
	"github.com/derekurban/forage/internal/credentials"
	"github.com/derekurban/forage/internal/providers"
	"github.com/derekurban/forage/internal/state"
)

type Router struct {
	Config   config.Config
	State    *state.Store
	Creds    credentials.Store
	Client   *http.Client
	Adapters map[string]Adapter
}

type Adapter interface {
	ID() string
}

type ProviderError struct {
	Code       string
	Message    string
	RetryAfter string
	ResetAt    string
	Observed   string
	HTTPStatus int
}

func (e ProviderError) Error() string { return e.Message }

func New(cfg config.Config, st *state.Store, creds credentials.Store) Router {
	r := Router{
		Config:   cfg,
		State:    st,
		Creds:    creds,
		Client:   &http.Client{Timeout: 25 * time.Second},
		Adapters: map[string]Adapter{},
	}
	for _, a := range DefaultAdapters(r.Client, creds) {
		r.Adapters[a.ID()] = a
	}
	return r
}

func (r Router) Search(ctx context.Context, req capability.SearchRequest) (capability.SearchResponse, *apperr.Error) {
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.CacheMode == "" {
		req.CacheMode = r.Config.Cache.Mode
	}
	key := cacheKey("search", req)
	var cached capability.SearchResponse
	ttl := time.Duration(r.Config.Cache.TTLHours) * time.Hour
	if req.CacheMode != "refresh" {
		ok, err := r.State.LatestRecord("search", key, ttl, &cached)
		if err == nil && ok {
			cached.CacheStatus = "hit"
			if !req.ExplainRouting {
				cached.Routing = nil
			}
			return cached, nil
		}
		if req.CacheMode == "only" {
			_ = r.State.PutNegativeCache(key, "cache_miss", 5*time.Minute)
			return capability.SearchResponse{}, apperr.New(apperr.CodeCacheMiss, "cache-only search missed", apperr.ExitCacheMiss)
		}
	}
	diag := capability.RoutingDiagnostics{Capability: req.Capability}
	var merged []capability.SearchResult
	for _, id := range r.eligible(req.Capability, req.Providers, req.ExcludeProviders, &diag) {
		a, ok := r.Adapters[id]
		if !ok {
			diag.Skipped = append(diag.Skipped, attempt(id, "skipped", "adapter_not_implemented"))
			continue
		}
		sp, ok := a.(capability.SearchProvider)
		if !ok {
			diag.Skipped = append(diag.Skipped, attempt(id, "skipped", "capability_not_supported"))
			continue
		}
		start := time.Now().UTC()
		results, err := sp.Search(ctx, req)
		aresult := attempt(id, "success", "")
		aresult.StartedAt = start.Format(time.RFC3339)
		aresult.FinishedAt = time.Now().UTC().Format(time.RFC3339)
		if err != nil {
			r.observeProviderError(id, err)
			aresult.Status = "failed"
			aresult.Reason = classify(err)
			diag.Attempts = append(diag.Attempts, aresult)
			_ = r.State.RecordAttempt(state.ProviderAttempt{Capability: req.Capability, Provider: id, Status: aresult.Status, Reason: aresult.Reason, StartedAt: aresult.StartedAt, FinishedAt: aresult.FinishedAt})
			continue
		}
		r.observeProviderSuccess(id)
		diag.Attempts = append(diag.Attempts, aresult)
		diag.ProvidersUsed = append(diag.ProvidersUsed, id)
		_ = r.State.RecordAttempt(state.ProviderAttempt{Capability: req.Capability, Provider: id, Status: "success", StartedAt: aresult.StartedAt, FinishedAt: aresult.FinishedAt})
		merged = append(merged, results...)
		if len(merged) >= req.Limit || len(diag.ProvidersUsed) >= maxFanout(r.Config.MaxFanout) {
			break
		}
	}
	merged = dedupe(merged)
	if len(merged) > req.Limit {
		merged = merged[:req.Limit]
	}
	if len(merged) == 0 {
		return capability.SearchResponse{}, exhausted(req.Capability, diag)
	}
	resp := capability.SearchResponse{Query: req.Query, Capability: req.Capability, Results: merged, Routing: &diag, CacheStatus: "miss"}
	_ = r.State.PutRecord("search", key, strings.Join(diag.ProvidersUsed, ","), "", req.Query, resp)
	if !req.ExplainRouting {
		resp.Routing = nil
	}
	return resp, nil
}

func (r Router) Fetch(ctx context.Context, req capability.FetchRequest) (capability.FetchResponse, *apperr.Error) {
	if req.CacheMode == "" {
		req.CacheMode = r.Config.Cache.Mode
	}
	key := cacheKey("fetch", req)
	var cached capability.FetchResponse
	if req.CacheMode != "refresh" {
		ok, err := r.State.LatestRecord("fetch", key, time.Duration(r.Config.Cache.TTLHours)*time.Hour, &cached)
		if err == nil && ok {
			cached.CacheStatus = "hit"
			if !req.ExplainRouting {
				cached.Routing = nil
			}
			return cached, nil
		}
		if req.CacheMode == "only" {
			_ = r.State.PutNegativeCache(key, "cache_miss", 5*time.Minute)
			return capability.FetchResponse{}, apperr.New(apperr.CodeCacheMiss, "cache-only fetch missed", apperr.ExitCacheMiss)
		}
	}
	diag := capability.RoutingDiagnostics{Capability: capability.FetchURL}
	for _, id := range r.eligible(capability.FetchURL, req.Providers, req.ExcludeProviders, &diag) {
		a, ok := r.Adapters[id]
		if !ok {
			diag.Skipped = append(diag.Skipped, attempt(id, "skipped", "adapter_not_implemented"))
			continue
		}
		fp, ok := a.(capability.FetchProvider)
		if !ok {
			diag.Skipped = append(diag.Skipped, attempt(id, "skipped", "capability_not_supported"))
			continue
		}
		doc, err := fp.Fetch(ctx, req)
		if err != nil {
			r.observeProviderError(id, err)
			diag.Attempts = append(diag.Attempts, attempt(id, "failed", classify(err)))
			_ = r.State.RecordAttempt(state.ProviderAttempt{Capability: capability.FetchURL, Provider: id, Status: "failed", Reason: classify(err)})
			continue
		}
		r.observeProviderSuccess(id)
		diag.Attempts = append(diag.Attempts, attempt(id, "success", ""))
		diag.ProvidersUsed = append(diag.ProvidersUsed, id)
		_ = r.State.RecordAttempt(state.ProviderAttempt{Capability: capability.FetchURL, Provider: id, Status: "success"})
		resp := capability.FetchResponse{Document: doc, Routing: &diag, CacheStatus: "miss"}
		_ = r.State.PutRecord("fetch", key, id, req.URL, doc.Title, resp)
		if !req.ExplainRouting {
			resp.Routing = nil
		}
		return resp, nil
	}
	return capability.FetchResponse{}, exhausted(capability.FetchURL, diag)
}

func (r Router) eligible(cap string, include, exclude []string, diag *capability.RoutingDiagnostics) []string {
	order := r.Config.Routing[cap]
	explicit := map[string]bool{}
	if len(include) > 0 {
		order = include
		for _, id := range include {
			explicit[id] = true
		}
	}
	var ids []string
	for _, id := range order {
		if slices.Contains(exclude, id) {
			diag.Skipped = append(diag.Skipped, attempt(id, "skipped", "excluded"))
			continue
		}
		if !config.Enabled(r.Config, id) && id != "direct" {
			diag.Skipped = append(diag.Skipped, attempt(id, "skipped", "disabled"))
			continue
		}
		p, ok := providers.ByID(id)
		authOK := true
		if ok {
			if !containsCapability(p.Capabilities, cap) {
				diag.Skipped = append(diag.Skipped, attempt(id, "skipped", "capability_not_supported"))
				continue
			}
			if (p.Status == providers.MetadataOnly || p.Status == providers.LegacyOptional) && !explicit[id] {
				diag.Skipped = append(diag.Skipped, attempt(id, "skipped", string(p.Status)))
				continue
			}
			var missing []string
			for _, f := range p.CredentialFields {
				if !f.Required {
					continue
				}
				cred, _ := r.Creds.GetField(id, f.Name, f.EnvVar)
				if !cred.Found {
					missing = append(missing, f.Name)
				}
			}
			if len(missing) > 0 {
				authOK = false
				diag.Skipped = append(diag.Skipped, attempt(id, "skipped", "auth_missing:"+strings.Join(missing, ",")))
				continue
			}
		}
		if ps, ok, err := r.State.ProviderState(id); err == nil && ok {
			if skip, reason := skipForState(ps, authOK); skip {
				diag.Skipped = append(diag.Skipped, attempt(id, "skipped", reason))
				continue
			}
		}
		ids = append(ids, id)
	}
	return ids
}

func (r Router) observeProviderError(id string, err error) {
	var pe ProviderError
	status := "degraded"
	reason := classify(err)
	retry := ""
	httpStatus := 0
	if errors.As(err, &pe) {
		retry = pe.RetryAfter
		httpStatus = pe.HTTPStatus
		if pe.Code == "rate_limited" {
			status = "cooldown"
		}
		if pe.Code == "auth_failed" {
			status = "disabled"
		}
	}
	var hs *int
	if httpStatus != 0 {
		hs = &httpStatus
	}
	reset := ""
	observed := ""
	if errors.As(err, &pe) {
		reset = pe.ResetAt
		observed = pe.Observed
	}
	_ = r.State.UpsertProviderState(state.ProviderState{Provider: id, Status: status, Reason: reason, RetryAfter: retry, ResetAt: reset, LastHTTPStatus: hs, LastCheckedAt: time.Now().UTC().Format(time.RFC3339), Observed: observed})
}

func (r Router) observeProviderSuccess(id string) {
	now := time.Now().UTC().Format(time.RFC3339)
	_ = r.State.UpsertProviderState(state.ProviderState{Provider: id, Status: "healthy", LastCheckedAt: now, LastSuccessAt: now})
}

func cacheKey(kind string, v any) string {
	b, _ := json.Marshal(v)
	sum := sha256.Sum256(append([]byte(kind+":"), b...))
	return hex.EncodeToString(sum[:])
}

func attempt(provider, status, reason string) capability.ProviderAttempt {
	now := time.Now().UTC().Format(time.RFC3339)
	return capability.ProviderAttempt{Provider: provider, Status: status, Reason: reason, StartedAt: now, FinishedAt: now}
}

func classify(err error) string {
	var pe ProviderError
	if errors.As(err, &pe) {
		return pe.Code
	}
	return "provider_error"
}

func exhausted(cap string, diag capability.RoutingDiagnostics) *apperr.Error {
	e := apperr.New(apperr.CodeCapabilityExhausted, fmt.Sprintf("No configured providers are currently available for %s.", cap), apperr.ExitCapabilityExhausted)
	e.Capability = cap
	for _, a := range append(diag.Attempts, diag.Skipped...) {
		e.Providers = append(e.Providers, apperr.ProviderInfo{Name: a.Provider, Status: a.Status, Reason: a.Reason})
	}
	return e
}

func maxFanout(v int) int {
	if v <= 0 {
		return 4
	}
	return v
}

func containsCapability(caps []string, cap string) bool {
	for _, c := range caps {
		if c == cap {
			return true
		}
	}
	return false
}

func skipForState(ps state.ProviderState, authOK bool) (bool, string) {
	switch ps.Status {
	case "cooldown", "exhausted":
		if retryTimeActive(ps.RetryAfter) || retryTimeActive(ps.ResetAt) || (ps.RetryAfter == "" && ps.ResetAt == "") {
			return true, ps.Status
		}
	case "disabled":
		if ps.Reason == "auth_missing" && authOK {
			return false, ""
		}
		return true, "disabled:" + ps.Reason
	}
	return false, ""
}

func retryTimeActive(v string) bool {
	if strings.TrimSpace(v) == "" {
		return false
	}
	if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
		return n > 0
	}
	if t, err := time.Parse(time.RFC1123, v); err == nil {
		return time.Now().UTC().Before(t)
	}
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return time.Now().UTC().Before(t)
	}
	return true
}

func dedupe(in []capability.SearchResult) []capability.SearchResult {
	seen := map[string]bool{}
	var out []capability.SearchResult
	for _, r := range in {
		key := strings.ToLower(strings.TrimSpace(r.CanonicalURL))
		if key == "" {
			key = strings.ToLower(strings.TrimSpace(r.URL))
		}
		if key == "" {
			key = strings.ToLower(r.Title + "|" + r.SourceDomain)
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, r)
	}
	return out
}
