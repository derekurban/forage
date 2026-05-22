package forage

import (
	"context"

	"github.com/derekurban/forage/internal/apperr"
	"github.com/derekurban/forage/internal/capability"
	"github.com/derekurban/forage/internal/config"
	"github.com/derekurban/forage/internal/credentials"
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
type ExtractedDocument = capability.ExtractedDocument
type RoutingDiagnostics = capability.RoutingDiagnostics
type Error = apperr.Error

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

func (c *Client) Search(ctx context.Context, req SearchRequest) (SearchResponse, *Error) {
	r := router.New(c.cfg, c.state, credentials.NewKeychainStore())
	return r.Search(ctx, req)
}

func (c *Client) Fetch(ctx context.Context, req FetchRequest) (FetchResponse, *Error) {
	r := router.New(c.cfg, c.state, credentials.NewKeychainStore())
	return r.Fetch(ctx, req)
}
