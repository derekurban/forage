package forage

import (
	"context"

	"github.com/derekurban/forage/internal/apperr"
	"github.com/derekurban/forage/internal/capability"
	"github.com/derekurban/forage/internal/config"
	"github.com/derekurban/forage/internal/credentials"
	"github.com/derekurban/forage/internal/doctor"
	"github.com/derekurban/forage/internal/evidence"
	"github.com/derekurban/forage/internal/providers"
	"github.com/derekurban/forage/internal/quota"
	"github.com/derekurban/forage/internal/router"
	"github.com/derekurban/forage/internal/state"
)

type Client struct {
	cfg   config.Config
	state *state.Store
}

type SearchRequest = capability.SearchRequest
type SearchResult = capability.SearchResult
type SearchResponse = capability.SearchResponse
type FetchRequest = capability.FetchRequest
type FetchResponse = capability.FetchResponse
type DataRequest = capability.DataRequest
type DataResponse = capability.DataResponse
type ExtractedDocument = capability.ExtractedDocument
type RoutingDiagnostics = capability.RoutingDiagnostics
type Error = apperr.Error
type ProviderState = state.ProviderState
type DoctorResult = doctor.Result
type EvidencePack = evidence.Pack

func Open() (*Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	st, err := state.Open(cfg.Cache.Database)
	if err != nil {
		return nil, err
	}
	return &Client{cfg: cfg, state: st}, nil
}

func (c *Client) Close() error {
	return c.state.Close()
}

func (c *Client) Scholar(ctx context.Context, req SearchRequest) (SearchResponse, *Error) {
	req.Capability = capability.SearchScholar
	r := router.New(c.cfg, c.state, credentials.NewKeychainStore())
	return r.Search(ctx, req)
}

func (c *Client) Fetch(ctx context.Context, req FetchRequest) (FetchResponse, *Error) {
	r := router.New(c.cfg, c.state, credentials.NewKeychainStore())
	return r.Fetch(ctx, req)
}

func (c *Client) Extract(ctx context.Context, req capability.ExtractRequest) (FetchResponse, *Error) {
	r := router.New(c.cfg, c.state, credentials.NewKeychainStore())
	return r.Fetch(ctx, FetchRequest{URL: req.URL, Providers: req.Providers, ExcludeProviders: req.ExcludeProviders, CacheMode: req.CacheMode, ExplainRouting: req.ExplainRouting})
}

func (c *Client) ArchiveLookup(ctx context.Context, req DataRequest) (DataResponse, *Error) {
	r := router.New(c.cfg, c.state, credentials.NewKeychainStore())
	return r.ArchiveLookup(ctx, req)
}

func (c *Client) EnrichDOI(ctx context.Context, req DataRequest) (DataResponse, *Error) {
	r := router.New(c.cfg, c.state, credentials.NewKeychainStore())
	return r.EnrichDOI(ctx, req)
}

func (c *Client) Citations(ctx context.Context, req DataRequest) (DataResponse, *Error) {
	r := router.New(c.cfg, c.state, credentials.NewKeychainStore())
	return r.Citations(ctx, req)
}

func (c *Client) Doctor(ctx context.Context) ([]DoctorResult, error) {
	d := doctor.New(credentials.NewKeychainStore(), c.state)
	var out []DoctorResult
	for _, p := range providers.Sorted() {
		out = append(out, d.Check(ctx, c.cfg, p))
	}
	return out, nil
}

func (c *Client) Quota() ([]ProviderState, error) {
	return c.state.ProviderStates()
}

func (c *Client) QuotaPreflight(ctx context.Context, provider string) quota.Result {
	return quota.New(c.cfg, c.state, credentials.NewKeychainStore()).Preflight(ctx, provider)
}

func (c *Client) CreateEvidencePack(query string, items []any) (string, EvidencePack, error) {
	return evidence.Create(config.Dir(), query, c.cfg.Policy, items)
}
