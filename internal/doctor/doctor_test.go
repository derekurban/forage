package doctor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/derekurban/forage/internal/config"
	"github.com/derekurban/forage/internal/credentials"
	"github.com/derekurban/forage/internal/providers"
	"github.com/derekurban/forage/internal/state"
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

type fakeCreds struct{}

func (fakeCreds) Get(provider, envVar string) (credentials.Credential, error) {
	return credentials.Credential{Found: true, Source: "test", Value: "x@example.com"}, nil
}
func (fakeCreds) Set(provider, value string) error { return nil }
func (fakeCreds) Delete(provider string) error     { return nil }
func (fakeCreds) GetField(provider, field, envVar string) (credentials.Credential, error) {
	return credentials.Credential{Found: true, Source: "test", Value: "x@example.com"}, nil
}
func (fakeCreds) SetField(provider, field, value string) error { return nil }
func (fakeCreds) DeleteField(provider, field string) error     { return nil }

func TestPromotedProviderDoctorProbes(t *testing.T) {
	for _, id := range []string{"semantic_scholar", "opencitations", "unpaywall"} {
		t.Run(id, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"ok":true}`))
			}))
			defer ts.Close()
			u, _ := url.Parse(ts.URL)
			st, err := state.Open(filepath.Join(t.TempDir(), "state.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer st.Close()
			r := New(fakeCreds{}, st)
			r.Client = &http.Client{Transport: rewriteTransport{base: u, rt: http.DefaultTransport}}
			p, ok := providers.ByID(id)
			if !ok {
				t.Fatalf("missing provider %s", id)
			}
			res := r.Check(context.Background(), config.Default(), p)
			if res.HealthStatus != "healthy" {
				t.Fatalf("health = %s message=%s", res.HealthStatus, res.Message)
			}
		})
	}
}
