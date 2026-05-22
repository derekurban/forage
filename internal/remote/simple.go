package remote

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	HTTP *http.Client
}

func New() Client {
	return Client{HTTP: &http.Client{Timeout: 20 * time.Second}}
}

func (c Client) GetJSON(ctx context.Context, endpoint string, headers map[string]string, dest any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return json.NewDecoder(resp.Body).Decode(dest)
}

func CrossrefDOI(ctx context.Context, doi string) (any, error) {
	var raw any
	err := New().GetJSON(ctx, "https://api.crossref.org/works/"+url.PathEscape(doi), map[string]string{"User-Agent": "forage/0.1"}, &raw)
	return raw, err
}

func OpenAlexWork(ctx context.Context, id string) (any, error) {
	var raw any
	endpoint := "https://api.openalex.org/works/" + url.PathEscape(id)
	if strings.HasPrefix(strings.ToLower(id), "10.") {
		endpoint = "https://api.openalex.org/works/doi:" + url.PathEscape(id)
	}
	err := New().GetJSON(ctx, endpoint, nil, &raw)
	return raw, err
}

func OpenCitations(ctx context.Context, doi string, direction string) (any, error) {
	var raw any
	if direction == "" {
		direction = "citations"
	}
	err := New().GetJSON(ctx, "https://opencitations.net/index/api/v1/"+direction+"/"+url.PathEscape(doi), nil, &raw)
	return raw, err
}

func CommonCrawlIndexes(ctx context.Context) (any, error) {
	var raw any
	err := New().GetJSON(ctx, "https://index.commoncrawl.org/collinfo.json", nil, &raw)
	return raw, err
}

func GDELTDocs(ctx context.Context, query string, limit int) (any, error) {
	if limit <= 0 {
		limit = 10
	}
	var raw any
	err := New().GetJSON(ctx, "https://api.gdeltproject.org/api/v2/doc/doc?query="+url.QueryEscape(query)+"&mode=artlist&format=json&maxrecords="+fmt.Sprint(limit), nil, &raw)
	return raw, err
}
