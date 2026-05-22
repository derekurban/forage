package quota

import (
	"context"
	"encoding/json"
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
	Provider string              `json:"provider"`
	OK       bool                `json:"ok"`
	Status   string              `json:"status"`
	Message  string              `json:"message,omitempty"`
	State    state.ProviderState `json:"state"`
}

type Service struct {
	Config config.Config
	State  *state.Store
	Creds  credentials.Store
	Client *http.Client
}

func New(cfg config.Config, st *state.Store, creds credentials.Store) Service {
	return Service{Config: cfg, State: st, Creds: creds, Client: &http.Client{Timeout: 15 * time.Second}}
}

func (s Service) Preflight(ctx context.Context, provider string) Result {
	p, ok := providers.ByID(provider)
	if !ok {
		return Result{Provider: provider, Status: "unknown_provider", Message: "provider is not registered"}
	}
	if !p.Quota.CanPreflight || strings.TrimSpace(p.Quota.Endpoint) == "" {
		return Result{Provider: provider, Status: "not_supported", Message: "provider has no quota preflight endpoint"}
	}
	switch p.ID {
	case "openalex":
		return s.preflightOpenAlex(ctx, p)
	default:
		return Result{Provider: p.ID, Status: "not_supported", Message: "provider preflight is not implemented"}
	}
}

func (s Service) preflightOpenAlex(ctx context.Context, p providers.Provider) Result {
	endpoint := p.Quota.Endpoint
	cred, _ := s.Creds.Get("openalex", "OPENALEX_API_KEY")
	if strings.TrimSpace(cred.Value) != "" {
		endpoint += "?api_key=" + url.QueryEscape(cred.Value)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Result{Provider: p.ID, Status: "unavailable", Message: err.Error()}
	}
	resp, err := s.Client.Do(req)
	if err != nil {
		return Result{Provider: p.ID, Status: "unavailable", Message: err.Error()}
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	ps := state.ProviderState{Provider: p.ID, Status: "healthy", LastCheckedAt: time.Now().UTC().Format(time.RFC3339), LastHTTPStatus: &resp.StatusCode, Observed: string(body)}
	if resp.StatusCode == http.StatusTooManyRequests {
		ps.Status = "cooldown"
		ps.Reason = "rate_limited"
		ps.RetryAfter = first(resp.Header.Get("Retry-After"))
		ps.ResetAt = firstHeader(resp.Header, "x-ratelimit-reset", "ratelimit-reset", "x-rate-limit-reset")
		_ = s.State.UpsertProviderState(ps)
		return Result{Provider: p.ID, Status: ps.Status, Message: "provider returned HTTP 429", State: ps}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		ps.Status = "degraded"
		ps.Reason = fmt.Sprintf("http_%d", resp.StatusCode)
		_ = s.State.UpsertProviderState(ps)
		return Result{Provider: p.ID, Status: ps.Status, Message: fmt.Sprintf("provider returned HTTP %d", resp.StatusCode), State: ps}
	}
	var raw map[string]any
	_ = json.Unmarshal(body, &raw)
	ps.Limit = findInt(raw, "limit", "daily_limit", "max_credits", "credits_limit")
	ps.Remaining = findInt(raw, "remaining", "daily_remaining", "credits_remaining")
	ps.Used = findInt(raw, "used", "daily_usage", "credits_used", "requests")
	if ps.Limit == nil {
		ps.Limit = headerInt(resp.Header, "x-ratelimit-limit", "ratelimit-limit", "x-rate-limit-limit")
	}
	if ps.Remaining == nil {
		ps.Remaining = headerInt(resp.Header, "x-ratelimit-remaining", "ratelimit-remaining", "x-rate-limit-remaining")
	}
	if ps.Used == nil {
		ps.Used = headerInt(resp.Header, "x-ratelimit-used", "x-ratelimit-credits-used", "ratelimit-used")
	}
	ps.ResetAt = firstHeader(resp.Header, "x-ratelimit-reset", "ratelimit-reset", "x-rate-limit-reset")
	_ = s.State.UpsertProviderState(ps)
	return Result{Provider: p.ID, OK: true, Status: ps.Status, State: ps}
}

func findInt(raw map[string]any, names ...string) *int64 {
	for _, name := range names {
		v, ok := raw[name]
		if !ok {
			continue
		}
		switch t := v.(type) {
		case float64:
			n := int64(t)
			return &n
		case string:
			n, err := strconv.ParseInt(strings.TrimSpace(t), 10, 64)
			if err == nil {
				return &n
			}
		}
	}
	return nil
}

func headerInt(h http.Header, names ...string) *int64 {
	for _, name := range names {
		v := first(h.Get(name))
		if v == "" {
			continue
		}
		n, err := strconv.ParseInt(v, 10, 64)
		if err == nil {
			return &n
		}
	}
	return nil
}

func firstHeader(h http.Header, names ...string) string {
	for _, name := range names {
		if v := first(h.Get(name)); v != "" {
			return v
		}
	}
	return ""
}

func first(v string) string {
	parts := strings.Split(v, ",")
	if len(parts) == 0 {
		return strings.TrimSpace(v)
	}
	return strings.TrimSpace(parts[0])
}
