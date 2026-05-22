package quota

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"

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

type rewriteTransport struct {
	base *url.URL
	rt   http.RoundTripper
}

func (t rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = t.base.Scheme
	req.URL.Host = t.base.Host
	return t.rt.RoundTrip(req)
}

func TestOpenAlexPreflightPersistsQuota(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Limit", "100")
		w.Header().Set("X-RateLimit-Remaining", "87")
		_, _ = w.Write([]byte(`{"daily_usage":13}`))
	}))
	defer ts.Close()
	st, err := state.Open(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	cfg := config.Default()
	svc := New(cfg, st, fakeCreds{})
	base, _ := url.Parse(ts.URL)
	svc.Client = &http.Client{Transport: rewriteTransport{base: base, rt: http.DefaultTransport}}
	res := svc.Preflight(context.Background(), "openalex")
	if !res.OK {
		t.Fatalf("result = %+v", res)
	}
	ps, ok, err := st.ProviderState("openalex")
	if err != nil || !ok {
		t.Fatalf("state = %+v/%v/%v", ps, ok, err)
	}
	if ps.Limit == nil || *ps.Limit != 100 || ps.Remaining == nil || *ps.Remaining != 87 || ps.Used == nil || *ps.Used != 13 {
		t.Fatalf("quota state = %+v", ps)
	}
}

func TestPreflightUnsupportedProvider(t *testing.T) {
	st, err := state.Open(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	res := New(config.Default(), st, fakeCreds{}).Preflight(context.Background(), "brave")
	if res.Status != "not_supported" {
		t.Fatalf("result = %+v", res)
	}
}
