package router

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/derekurban/forage/internal/capability"
	"github.com/derekurban/forage/internal/credentials"
)

type HTTPAdapter struct {
	id     string
	client *http.Client
	creds  credentials.Store
}

func (a HTTPAdapter) ID() string { return a.id }

func DefaultAdapters(client *http.Client, creds credentials.Store) []Adapter {
	ids := []string{
		"jina", "browserbase", "direct", "firecrawl", "scrapingant",
		"openalex", "semantic_scholar", "crossref", "arxiv", "pubmed", "datacite", "europepmc", "doaj",
		"opencitations", "unpaywall",
		"internet_archive", "commoncrawl",
	}
	var out []Adapter
	for _, id := range ids {
		out = append(out, HTTPAdapter{id: id, client: client, creds: creds})
	}
	return out
}

func (a HTTPAdapter) Search(ctx context.Context, req capability.SearchRequest) ([]capability.SearchResult, error) {
	switch req.Capability {
	case capability.SearchWeb:
		return a.searchWeb(ctx, req)
	case capability.SearchNews:
		return a.searchNews(ctx, req)
	case capability.SearchPlatform:
		return a.searchPlatform(ctx, req)
	case capability.SearchScholar:
		works, err := a.searchScholar(ctx, req)
		if err != nil {
			return nil, err
		}
		var out []capability.SearchResult
		for i, w := range works {
			out = append(out, capability.SearchResult{URL: w.URL, Title: w.Title, Snippet: w.Abstract, Provider: w.Provider, ProviderRank: i + 1, ResultType: "paper", Raw: w.Raw})
		}
		return out, nil
	default:
		return nil, ProviderError{Code: "unsupported_capability", Message: "unsupported search capability"}
	}
}

func (a HTTPAdapter) Fetch(ctx context.Context, req capability.FetchRequest) (capability.ExtractedDocument, error) {
	switch a.id {
	case "direct":
		return a.directFetch(ctx, req.URL)
	case "jina":
		return a.jinaFetch(ctx, req.URL)
	case "browserbase":
		return a.browserbaseFetch(ctx, req.URL)
	case "firecrawl":
		return a.firecrawlExtract(ctx, req.URL)
	case "scrapingant":
		return a.scrapingAntExtract(ctx, req.URL)
	default:
		return capability.ExtractedDocument{}, ProviderError{Code: "unsupported_capability", Message: "fetch unsupported by provider"}
	}
}

func (a HTTPAdapter) Extract(ctx context.Context, req capability.ExtractRequest) (capability.ExtractedDocument, error) {
	return a.Fetch(ctx, capability.FetchRequest{URL: req.URL, CacheMode: req.CacheMode, ExplainRouting: req.ExplainRouting})
}

func (a HTTPAdapter) LookupArchive(ctx context.Context, target string, limit int) ([]capability.ArchiveRecord, error) {
	switch a.id {
	case "internet_archive":
		var raw struct {
			ArchivedSnapshots struct {
				Closest struct {
					Available bool   `json:"available"`
					URL       string `json:"url"`
					Timestamp string `json:"timestamp"`
					Status    string `json:"status"`
				} `json:"closest"`
			} `json:"archived_snapshots"`
		}
		u := "https://archive.org/wayback/available?url=" + url.QueryEscape(target)
		if err := a.getJSON(ctx, u, nil, &raw); err != nil {
			return nil, err
		}
		c := raw.ArchivedSnapshots.Closest
		if !c.Available && c.URL == "" {
			return nil, ProviderError{Code: "not_found", Message: "no archived snapshot found"}
		}
		return []capability.ArchiveRecord{{URL: target, ArchiveURL: c.URL, Timestamp: c.Timestamp, Status: c.Status, Provider: a.id}}, nil
	case "commoncrawl":
		var indexes []struct {
			ID       string `json:"id"`
			APIURL   string `json:"cdx-api"`
			Name     string `json:"name"`
			Timegate string `json:"timegate"`
		}
		if err := a.getJSON(ctx, "https://index.commoncrawl.org/collinfo.json", nil, &indexes); err != nil {
			return nil, err
		}
		if len(indexes) == 0 {
			return nil, ProviderError{Code: "not_found", Message: "no Common Crawl indexes found"}
		}
		max := limit
		if max <= 0 || max > len(indexes) {
			max = len(indexes)
		}
		var out []capability.ArchiveRecord
		for _, idx := range indexes[:max] {
			if strings.TrimSpace(idx.APIURL) == "" {
				continue
			}
			u := idx.APIURL + "?url=" + url.QueryEscape(target) + "&output=json&limit=1"
			txt, err := a.getText(ctx, u, nil)
			if err != nil {
				continue
			}
			for _, line := range strings.Split(txt, "\n") {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				var raw struct {
					URL       string `json:"url"`
					Timestamp string `json:"timestamp"`
					Status    string `json:"status"`
					Mime      string `json:"mime"`
					Digest    string `json:"digest"`
				}
				if err := json.Unmarshal([]byte(line), &raw); err != nil {
					continue
				}
				seenURL := raw.URL
				if seenURL == "" {
					seenURL = target
				}
				status := raw.Status
				if status == "" {
					status = idx.Name
				}
				out = append(out, capability.ArchiveRecord{URL: seenURL, ArchiveURL: idx.APIURL, Timestamp: raw.Timestamp, Status: status, MimeType: raw.Mime, Provider: a.id})
				break
			}
			if len(out) >= max {
				break
			}
		}
		if len(out) == 0 {
			return nil, ProviderError{Code: "not_found", Message: "target URL was not found in recent Common Crawl indexes"}
		}
		return out, nil
	default:
		return nil, ProviderError{Code: "unsupported_capability", Message: "archive lookup unsupported"}
	}
}

func (a HTTPAdapter) Enrich(ctx context.Context, req capability.DataRequest) (any, error) {
	switch req.Capability {
	case capability.EnrichDOI:
		return a.enrichDOI(ctx, req)
	case capability.EnrichPaper:
		return a.enrichPaper(ctx, req)
	case capability.EnrichAuthor:
		return a.enrichAuthor(ctx, req.ID)
	default:
		return nil, ProviderError{Code: "unsupported_capability", Message: "enrichment unsupported"}
	}
}

func (a HTTPAdapter) Citations(ctx context.Context, req capability.DataRequest) (any, error) {
	doi := req.ID
	if doi == "" {
		doi = req.Query
	}
	switch a.id {
	case "opencitations":
		token, _ := a.creds.Get("opencitations", "OPENCITATIONS_ACCESS_TOKEN")
		headers := map[string]string{}
		if strings.TrimSpace(token.Value) != "" {
			headers["authorization"] = token.Value
		}
		var raw []map[string]any
		if err := a.getJSON(ctx, "https://opencitations.net/index/api/v1/citations/"+url.PathEscape(doi), headers, &raw); err != nil {
			return nil, err
		}
		records := make([]capability.CitationRecord, 0, len(raw))
		for _, item := range raw {
			rec := capability.CitationRecord{
				CitingDOI: stringField(item, "citing"),
				CitedDOI:  stringField(item, "cited"),
				Date:      stringField(item, "creation"),
				Provider:  a.id,
			}
			rec.Year = yearFromDate(rec.Date)
			if req.IncludeRaw {
				rec.Raw = item
			}
			records = append(records, rec)
		}
		resp := capability.CitationResponse{DOI: doi, Records: records, Provider: a.id, Summary: citationSummary(a.id, len(records), 0, "OpenCitations returns citation links present in its indexed citation corpus; sparse results mean limited provider coverage, not necessarily no citations.")}
		if req.IncludeRaw {
			resp.Raw = raw
		}
		return resp, nil
	case "crossref":
		raw, err := a.enrichDOI(ctx, capability.DataRequest{ID: doi, IncludeRaw: req.IncludeRaw})
		if err != nil {
			return nil, err
		}
		return capability.CitationResponse{DOI: doi, Provider: a.id, Summary: citationSummary(a.id, 0, 0, "Crossref is used as fallback metadata only here; it does not expose citing-work records through this route."), Raw: includeRaw(req.IncludeRaw, raw)}, nil
	case "openalex":
		key, _ := a.creds.Get("openalex", "OPENALEX_API_KEY")
		lookup := "https://api.openalex.org/works/doi:" + url.PathEscape(cleanDOI(doi))
		if key.Value != "" {
			lookup += "?api_key=" + url.QueryEscape(key.Value)
		}
		var work map[string]any
		if err := a.getJSON(ctx, lookup, nil, &work); err != nil {
			return nil, err
		}
		workID := strings.TrimPrefix(stringField(work, "id"), "https://openalex.org/")
		if workID == "" {
			return nil, ProviderError{Code: "not_found", Message: "OpenAlex work ID not found for DOI"}
		}
		u := "https://api.openalex.org/works?filter=cites:" + url.QueryEscape(workID) + "&per-page=" + fmt.Sprint(limit(req.Limit))
		if key.Value != "" {
			u += "&api_key=" + url.QueryEscape(key.Value)
		}
		var raw struct {
			Results []map[string]any `json:"results"`
			Meta    map[string]any   `json:"meta"`
		}
		if err := a.getJSON(ctx, u, nil, &raw); err != nil {
			return nil, err
		}
		records := make([]capability.CitationRecord, 0, len(raw.Results))
		for _, item := range raw.Results {
			rec := capability.CitationRecord{CitingDOI: stringField(item, "doi"), CitedDOI: doi, Title: stringField(item, "title", "display_name"), URL: nestedString(item, "primary_location", "landing_page_url"), Year: intField(item, "publication_year"), Provider: a.id}
			if req.IncludeRaw {
				rec.Raw = item
			}
			records = append(records, rec)
		}
		total := intField(raw.Meta, "count")
		resp := capability.CitationResponse{DOI: doi, Records: records, Provider: a.id, Summary: citationSummary(a.id, len(records), total, "OpenAlex returns works matching cites:<work>; returned records are limited by the command limit and OpenAlex coverage.")}
		resp.Summary["openalex_work"] = workID
		if req.IncludeRaw {
			resp.Raw = raw
		}
		return resp, nil
	case "semantic_scholar":
		paperID := doi
		if looksLikeDOI(doi) {
			paperID = "DOI:" + cleanDOI(doi)
		}
		var raw struct {
			Data []struct {
				CitingPaper semanticScholarPaper `json:"citingPaper"`
			} `json:"data"`
		}
		u := "https://api.semanticscholar.org/graph/v1/paper/" + url.PathEscape(paperID) + "/citations?limit=" + fmt.Sprint(limit(req.Limit)) + "&fields=title,url,year,externalIds"
		if err := a.getJSON(ctx, u, nil, &raw); err != nil {
			return nil, err
		}
		records := make([]capability.CitationRecord, 0, len(raw.Data))
		for _, item := range raw.Data {
			rec := capability.CitationRecord{CitingDOI: item.CitingPaper.ExternalIDs.DOI, CitedDOI: doi, Title: item.CitingPaper.Title, URL: item.CitingPaper.URL, Year: item.CitingPaper.Year, Provider: a.id}
			if req.IncludeRaw {
				rec.Raw = item
			}
			records = append(records, rec)
		}
		resp := capability.CitationResponse{DOI: doi, Records: records, Provider: a.id, Summary: citationSummary(a.id, len(records), 0, "Semantic Scholar returns citing papers available in its graph; public unauthenticated access may be rate-limited and coverage varies by work.")}
		if req.IncludeRaw {
			resp.Raw = raw
		}
		return resp, nil
	default:
		return nil, ProviderError{Code: "unsupported_capability", Message: "citations unsupported"}
	}
}

func (a HTTPAdapter) Corpus(ctx context.Context, req capability.DataRequest) (any, error) {
	switch a.id {
	case "commoncrawl":
		var raw any
		if err := a.getJSON(ctx, "https://index.commoncrawl.org/collinfo.json", nil, &raw); err != nil {
			return nil, err
		}
		return map[string]any{"provider": a.id, "query": req.Query, "indexes": raw}, nil
	case "gdelt":
		return a.searchNews(ctx, capability.SearchRequest{Query: req.Query, Capability: capability.SearchNews, Limit: req.Limit})
	case "internet_archive":
		return map[string]any{"provider": a.id, "query": req.Query, "status": "use archive lookup URL for URL-specific Wayback availability"}, nil
	default:
		return nil, ProviderError{Code: "unsupported_capability", Message: "corpus query unsupported"}
	}
}

func (a HTTPAdapter) Render(ctx context.Context, req capability.FetchRequest) (capability.ExtractedDocument, error) {
	switch a.id {
	case "scrapingant", "browserbase":
		doc, err := a.Fetch(ctx, req)
		if err != nil {
			return capability.ExtractedDocument{}, err
		}
		doc.ExtractionMethod = "render_" + doc.ExtractionMethod
		return doc, nil
	default:
		return capability.ExtractedDocument{}, ProviderError{Code: "unsupported_capability", Message: "browser rendering unsupported"}
	}
}

func (a HTTPAdapter) Crawl(ctx context.Context, target string, maxPages int) ([]capability.ExtractedDocument, error) {
	if maxPages <= 0 {
		maxPages = 10
	}
	switch a.id {
	case "direct":
		return a.localCrawl(ctx, target, maxPages)
	case "firecrawl":
		key, _ := a.creds.Get("firecrawl", "FIRECRAWL_API_KEY")
		body := map[string]any{"url": target, "limit": maxPages, "scrapeOptions": map[string]any{"formats": []string{"markdown", "html"}}}
		var raw struct {
			Success bool `json:"success"`
			Data    []struct {
				Markdown string         `json:"markdown"`
				HTML     string         `json:"html"`
				Metadata map[string]any `json:"metadata"`
			} `json:"data"`
		}
		if err := a.postJSON(ctx, "https://api.firecrawl.dev/v1/crawl", map[string]string{"Authorization": "Bearer " + key.Value}, body, &raw); err != nil {
			return nil, err
		}
		var out []capability.ExtractedDocument
		for _, item := range raw.Data {
			u, _ := item.Metadata["sourceURL"].(string)
			md := item.Markdown
			out = append(out, capability.ExtractedDocument{URL: u, RetrievedAt: time.Now().UTC(), Markdown: md, HTML: item.HTML, PlainText: stripMarkdown(md), Provider: a.id, ExtractionMethod: "firecrawl_crawl", QualityScore: quality(md), ContentHash: hash(md), Raw: item.Metadata})
		}
		if len(out) == 0 {
			return nil, ProviderError{Code: "not_found", Message: "crawl returned no documents"}
		}
		return out, nil
	default:
		return nil, ProviderError{Code: "unsupported_capability", Message: "crawl unsupported"}
	}
}

func (a HTTPAdapter) enrichDOI(ctx context.Context, req capability.DataRequest) (any, error) {
	doi := req.ID
	switch a.id {
	case "crossref":
		var raw map[string]any
		if err := a.getJSON(ctx, "https://api.crossref.org/works/"+url.PathEscape(doi), map[string]string{"User-Agent": "forage/0.1"}, &raw); err != nil {
			return nil, err
		}
		return capability.ScholarlyEnrichment{ID: doi, DOI: doi, Provider: a.id, Record: normalizeCrossrefWork(raw, a.id, req.IncludeRaw), Raw: includeRaw(req.IncludeRaw, raw)}, nil
	case "openalex":
		return a.enrichPaper(ctx, capability.DataRequest{ID: "doi:" + doi, Query: doi, IncludeRaw: req.IncludeRaw})
	case "unpaywall":
		email, _ := a.creds.GetField("unpaywall", "email", "UNPAYWALL_EMAIL")
		var raw map[string]any
		u := "https://api.unpaywall.org/v2/" + url.PathEscape(doi) + "?email=" + url.QueryEscape(email.Value)
		if err := a.getJSON(ctx, u, nil, &raw); err != nil {
			return nil, err
		}
		return capability.ScholarlyEnrichment{ID: doi, DOI: doi, Provider: a.id, OpenAccess: normalizeUnpaywall(raw, a.id, req.IncludeRaw), Raw: includeRaw(req.IncludeRaw, raw)}, nil
	default:
		return nil, ProviderError{Code: "unsupported_capability", Message: "DOI enrichment unsupported"}
	}
}

func (a HTTPAdapter) enrichPaper(ctx context.Context, req capability.DataRequest) (any, error) {
	id := req.ID
	switch a.id {
	case "openalex":
		key, _ := a.creds.Get("openalex", "OPENALEX_API_KEY")
		endpoint := "https://api.openalex.org/works/" + url.PathEscape(id)
		if strings.HasPrefix(strings.ToLower(id), "10.") {
			endpoint = "https://api.openalex.org/works/doi:" + url.PathEscape(id)
		}
		if key.Value != "" {
			sep := "?"
			if strings.Contains(endpoint, "?") {
				sep = "&"
			}
			endpoint += sep + "api_key=" + url.QueryEscape(key.Value)
		}
		var raw map[string]any
		if err := a.getJSON(ctx, endpoint, nil, &raw); err != nil {
			return nil, err
		}
		return capability.ScholarlyEnrichment{ID: id, DOI: stringField(raw, "doi"), Provider: a.id, Record: normalizeOpenAlexWork(raw, a.id, req.IncludeRaw), OpenAccess: normalizeOpenAlexOA(raw, a.id, req.IncludeRaw), Raw: includeRaw(req.IncludeRaw, raw)}, nil
	case "semantic_scholar":
		paperID := id
		if looksLikeDOI(id) {
			paperID = "DOI:" + cleanDOI(id)
		}
		var raw semanticScholarPaper
		u := "https://api.semanticscholar.org/graph/v1/paper/" + url.PathEscape(paperID) + "?fields=paperId,title,abstract,url,year,externalIds,citationCount,referenceCount,authors,venue"
		if err := a.getJSON(ctx, u, nil, &raw); err != nil {
			return nil, err
		}
		return capability.ScholarlyEnrichment{ID: id, DOI: raw.ExternalIDs.DOI, Provider: a.id, Record: normalizeSemanticScholarPaper(raw, a.id, req.IncludeRaw), Raw: includeRaw(req.IncludeRaw, raw)}, nil
	default:
		return nil, ProviderError{Code: "unsupported_capability", Message: "paper enrichment unsupported"}
	}
}

func (a HTTPAdapter) enrichAuthor(ctx context.Context, id string) (any, error) {
	switch a.id {
	case "orcid":
		return map[string]any{"provider": a.id, "orcid": id, "url": "https://orcid.org/" + id, "status": "link_only"}, nil
	case "openalex":
		var raw any
		if err := a.getJSON(ctx, "https://api.openalex.org/authors/"+url.PathEscape(id), nil, &raw); err != nil {
			return nil, err
		}
		return map[string]any{"provider": a.id, "id": id, "author": raw}, nil
	default:
		return nil, ProviderError{Code: "unsupported_capability", Message: "author enrichment unsupported"}
	}
}

func (a HTTPAdapter) searchWeb(ctx context.Context, req capability.SearchRequest) ([]capability.SearchResult, error) {
	q := req.Query
	if req.Site != "" {
		q += " site:" + req.Site
	}
	switch a.id {
	case "brave":
		key, _ := a.creds.GetField("brave", "search_api_key", "BRAVE_SEARCH_API_KEY")
		u := "https://api.search.brave.com/res/v1/web/search?q=" + url.QueryEscape(q) + "&count=" + fmt.Sprint(limit(req.Limit))
		h := map[string]string{"X-Subscription-Token": key.Value, "Accept": "application/json"}
		var raw struct {
			Web struct {
				Results []struct{ Title, URL, Description string } `json:"results"`
			} `json:"web"`
		}
		if err := a.getJSON(ctx, u, h, &raw); err != nil {
			return nil, err
		}
		var out []capability.SearchResult
		for i, r := range raw.Web.Results {
			out = append(out, result(a.id, i, r.URL, r.Title, r.Description, "web"))
		}
		return out, nil
	case "jina":
		key, _ := a.creds.Get("jina", "JINA_API_KEY")
		headers := bearerHeaders(key.Value)
		if headers == nil {
			headers = map[string]string{}
		}
		headers["Accept"] = "application/json"
		u := "https://s.jina.ai/" + url.QueryEscape(q)
		var raw struct {
			Data []struct {
				Title, URL, Description, Content string
			} `json:"data"`
		}
		err := a.getJSON(ctx, u, headers, &raw)
		if err != nil {
			txt, terr := a.getText(ctx, u, headers)
			if terr != nil {
				return nil, err
			}
			return parseMarkdownLinks(a.id, txt, "web"), nil
		}
		var out []capability.SearchResult
		for i, r := range raw.Data {
			snippet := r.Description
			if snippet == "" {
				snippet = r.Content
			}
			out = append(out, result(a.id, i, r.URL, r.Title, snippet, "web"))
		}
		return out, nil
	case "browserbase":
		key, _ := a.creds.Get("browserbase", "BROWSERBASE_API_KEY")
		body := map[string]any{"query": q, "numResults": min(limit(req.Limit), 25)}
		var raw struct {
			Results []struct {
				Title, URL, Author, PublishedDate, Image, Favicon string
			} `json:"results"`
		}
		if err := a.postJSON(ctx, "https://api.browserbase.com/v1/search", map[string]string{"x-bb-api-key": key.Value}, body, &raw); err != nil {
			return nil, err
		}
		var out []capability.SearchResult
		for i, r := range raw.Results {
			rr := result(a.id, i, r.URL, r.Title, r.Author, "web")
			rr.SourceName = r.Author
			out = append(out, rr)
		}
		return out, nil
	case "tavily":
		key, _ := a.creds.Get("tavily", "TAVILY_API_KEY")
		body := map[string]any{"query": q, "max_results": limit(req.Limit)}
		var raw struct {
			Results []struct {
				Title, URL, Content string
				Score               float64
			} `json:"results"`
		}
		if err := a.postJSON(ctx, "https://api.tavily.com/search", map[string]string{"Authorization": "Bearer " + key.Value}, body, &raw); err != nil {
			return nil, err
		}
		var out []capability.SearchResult
		for i, r := range raw.Results {
			rr := result(a.id, i, r.URL, r.Title, r.Content, "web")
			rr.Score = r.Score
			out = append(out, rr)
		}
		return out, nil
	case "exa":
		key, _ := a.creds.Get("exa", "EXA_API_KEY")
		body := map[string]any{"query": q, "numResults": limit(req.Limit)}
		var raw struct {
			Results []struct {
				Title, URL, Text string
				Score            float64
				PublishedDate    string `json:"publishedDate"`
			} `json:"results"`
		}
		if err := a.postJSON(ctx, "https://api.exa.ai/search", map[string]string{"x-api-key": key.Value}, body, &raw); err != nil {
			return nil, err
		}
		var out []capability.SearchResult
		for i, r := range raw.Results {
			rr := result(a.id, i, r.URL, r.Title, r.Text, "web")
			rr.Score = r.Score
			out = append(out, rr)
		}
		return out, nil
	case "serpapi":
		key, _ := a.creds.Get("serpapi", "SERPAPI_API_KEY")
		u := "https://serpapi.com/search.json?engine=google&q=" + url.QueryEscape(q) + "&num=" + fmt.Sprint(limit(req.Limit)) + "&api_key=" + url.QueryEscape(key.Value)
		var raw struct {
			Organic []struct{ Title, Link, Snippet string } `json:"organic_results"`
		}
		if err := a.getJSON(ctx, u, nil, &raw); err != nil {
			return nil, err
		}
		var out []capability.SearchResult
		for i, r := range raw.Organic {
			out = append(out, result(a.id, i, r.Link, r.Title, r.Snippet, "web"))
		}
		return out, nil
	case "serpstack":
		key, _ := a.creds.Get("serpstack", "SERPSTACK_API_KEY")
		u := "http://api.serpstack.com/search?access_key=" + url.QueryEscape(key.Value) + "&query=" + url.QueryEscape(q) + "&num=" + fmt.Sprint(limit(req.Limit))
		var raw struct {
			Organic []struct{ Title, URL, Snippet string } `json:"organic_results"`
		}
		if err := a.getJSON(ctx, u, nil, &raw); err != nil {
			return nil, err
		}
		var out []capability.SearchResult
		for i, r := range raw.Organic {
			out = append(out, result(a.id, i, r.URL, r.Title, r.Snippet, "web"))
		}
		return out, nil
	default:
		return nil, ProviderError{Code: "unsupported_capability", Message: "web search unsupported"}
	}
}

func (a HTTPAdapter) searchNews(ctx context.Context, req capability.SearchRequest) ([]capability.SearchResult, error) {
	q := url.QueryEscape(req.Query)
	switch a.id {
	case "gdelt":
		u := "https://api.gdeltproject.org/api/v2/doc/doc?query=" + q + "&mode=artlist&format=json&maxrecords=" + fmt.Sprint(limit(req.Limit))
		var raw struct {
			Articles []struct{ Title, URL, SourceCountry, Domain, Seendate string } `json:"articles"`
		}
		if err := a.getJSON(ctx, u, nil, &raw); err != nil {
			return nil, err
		}
		var out []capability.SearchResult
		for i, r := range raw.Articles {
			rr := result(a.id, i, r.URL, r.Title, r.Domain, "news")
			rr.SourceName = r.Domain
			out = append(out, rr)
		}
		return out, nil
	case "brave":
		key, _ := a.creds.GetField("brave", "search_api_key", "BRAVE_SEARCH_API_KEY")
		u := "https://api.search.brave.com/res/v1/news/search?q=" + q + "&count=" + fmt.Sprint(limit(req.Limit))
		h := map[string]string{"X-Subscription-Token": key.Value, "Accept": "application/json"}
		var raw struct {
			Results []struct {
				Title, URL, Description, Age string
			} `json:"results"`
		}
		if err := a.getJSON(ctx, u, h, &raw); err != nil {
			return nil, err
		}
		var out []capability.SearchResult
		for i, r := range raw.Results {
			out = append(out, result(a.id, i, r.URL, r.Title, r.Description, "news"))
		}
		return out, nil
	case "guardian":
		key, _ := a.creds.Get("guardian", "GUARDIAN_API_KEY")
		u := "https://content.guardianapis.com/search?q=" + q + "&page-size=" + fmt.Sprint(limit(req.Limit)) + "&api-key=" + url.QueryEscape(key.Value) + "&show-fields=trailText"
		var raw struct {
			Response struct {
				Results []struct {
					WebTitle, WebURL, WebPublicationDate string
					Fields                               struct{ TrailText string }
				} `json:"results"`
			} `json:"response"`
		}
		if err := a.getJSON(ctx, u, nil, &raw); err != nil {
			return nil, err
		}
		var out []capability.SearchResult
		for i, r := range raw.Response.Results {
			out = append(out, result(a.id, i, r.WebURL, r.WebTitle, stripHTML(r.Fields.TrailText), "news"))
		}
		return out, nil
	case "gnews":
		key, _ := a.creds.Get("gnews", "GNEWS_API_KEY")
		u := "https://gnews.io/api/v4/search?q=" + q + "&max=" + fmt.Sprint(limit(req.Limit)) + "&apikey=" + url.QueryEscape(key.Value)
		var raw struct {
			Articles []struct{ Title, Description, URL string } `json:"articles"`
		}
		if err := a.getJSON(ctx, u, nil, &raw); err != nil {
			return nil, err
		}
		var out []capability.SearchResult
		for i, r := range raw.Articles {
			out = append(out, result(a.id, i, r.URL, r.Title, r.Description, "news"))
		}
		return out, nil
	case "newsapi":
		key, _ := a.creds.Get("newsapi", "NEWSAPI_API_KEY")
		u := "https://newsapi.org/v2/everything?q=" + q + "&pageSize=" + fmt.Sprint(limit(req.Limit)) + "&apiKey=" + url.QueryEscape(key.Value)
		var raw struct {
			Articles []struct{ Title, Description, URL string } `json:"articles"`
		}
		if err := a.getJSON(ctx, u, nil, &raw); err != nil {
			return nil, err
		}
		var out []capability.SearchResult
		for i, r := range raw.Articles {
			out = append(out, result(a.id, i, r.URL, r.Title, r.Description, "news"))
		}
		return out, nil
	case "currents":
		key, _ := a.creds.Get("currents", "CURRENTS_API_KEY")
		u := "https://api.currentsapi.services/v1/search?keywords=" + q + "&page_size=" + fmt.Sprint(limit(req.Limit)) + "&apiKey=" + url.QueryEscape(key.Value)
		var raw struct {
			News []struct{ Title, Description, URL string } `json:"news"`
		}
		if err := a.getJSON(ctx, u, nil, &raw); err != nil {
			return nil, err
		}
		var out []capability.SearchResult
		for i, r := range raw.News {
			out = append(out, result(a.id, i, r.URL, r.Title, r.Description, "news"))
		}
		return out, nil
	case "mediastack":
		key, _ := a.creds.Get("mediastack", "MEDIASTACK_API_KEY")
		u := "http://api.mediastack.com/v1/news?access_key=" + url.QueryEscape(key.Value) + "&keywords=" + q + "&limit=" + fmt.Sprint(limit(req.Limit))
		var raw struct {
			Data []struct{ Title, Description, URL string } `json:"data"`
		}
		if err := a.getJSON(ctx, u, nil, &raw); err != nil {
			return nil, err
		}
		var out []capability.SearchResult
		for i, r := range raw.Data {
			out = append(out, result(a.id, i, r.URL, r.Title, r.Description, "news"))
		}
		return out, nil
	case "worldnews":
		key, _ := a.creds.Get("worldnews", "WORLDNEWS_API_KEY")
		u := "https://api.worldnewsapi.com/search-news?text=" + q + "&number=" + fmt.Sprint(limit(req.Limit))
		var raw struct {
			News []struct{ Title, Text, URL string } `json:"news"`
		}
		if err := a.getJSON(ctx, u, map[string]string{"x-api-key": key.Value}, &raw); err != nil {
			return nil, err
		}
		var out []capability.SearchResult
		for i, r := range raw.News {
			out = append(out, result(a.id, i, r.URL, r.Title, r.Text, "news"))
		}
		return out, nil
	default:
		return a.searchWeb(ctx, req)
	}
}

func (a HTTPAdapter) searchPlatform(ctx context.Context, req capability.SearchRequest) ([]capability.SearchResult, error) {
	switch a.id {
	case "hackernews":
		u := "https://hn.algolia.com/api/v1/search?query=" + url.QueryEscape(req.Query) + "&hitsPerPage=" + fmt.Sprint(limit(req.Limit))
		var raw struct {
			Hits []struct {
				Title, URL, ObjectID string
				StoryText            string `json:"story_text"`
			} `json:"hits"`
		}
		if err := a.getJSON(ctx, u, nil, &raw); err != nil {
			return nil, err
		}
		var out []capability.SearchResult
		for i, r := range raw.Hits {
			link := r.URL
			if link == "" {
				link = "https://news.ycombinator.com/item?id=" + r.ObjectID
			}
			out = append(out, result(a.id, i, link, r.Title, r.StoryText, "discussion"))
		}
		return out, nil
	case "forem":
		u := "https://dev.to/api/articles?tag=" + url.QueryEscape(req.Query) + "&per_page=" + fmt.Sprint(limit(req.Limit))
		var raw []struct{ Title, URL, Description string }
		if err := a.getJSON(ctx, u, nil, &raw); err != nil {
			return nil, err
		}
		var out []capability.SearchResult
		for i, r := range raw {
			out = append(out, result(a.id, i, r.URL, r.Title, r.Description, "article"))
		}
		return out, nil
	default:
		return nil, ProviderError{Code: "unsupported_capability", Message: "platform search unsupported"}
	}
}

func (a HTTPAdapter) searchScholar(ctx context.Context, req capability.SearchRequest) ([]capability.ScholarWork, error) {
	q := url.QueryEscape(req.Query)
	switch a.id {
	case "openalex":
		key, _ := a.creds.Get("openalex", "OPENALEX_API_KEY")
		u := "https://api.openalex.org/works?search=" + q + "&per-page=" + fmt.Sprint(limit(req.Limit))
		if key.Value != "" {
			u += "&api_key=" + url.QueryEscape(key.Value)
		}
		var raw struct {
			Results []struct {
				ID, DOI, Title  string
				PublicationYear int `json:"publication_year"`
				PrimaryLocation struct {
					LandingPageURL string `json:"landing_page_url"`
				} `json:"primary_location"`
			} `json:"results"`
		}
		if err := a.getJSON(ctx, u, nil, &raw); err != nil {
			return nil, err
		}
		var out []capability.ScholarWork
		for _, r := range raw.Results {
			out = append(out, capability.ScholarWork{ID: r.ID, DOI: r.DOI, Title: r.Title, Year: r.PublicationYear, URL: r.PrimaryLocation.LandingPageURL, Provider: a.id})
		}
		return out, nil
	case "crossref":
		var raw struct {
			Message struct {
				Items []struct {
					DOI            string
					Title          []string
					URL            string
					PublishedPrint struct {
						DateParts [][]int `json:"date-parts"`
					} `json:"published-print"`
				} `json:"items"`
			} `json:"message"`
		}
		if err := a.getJSON(ctx, "https://api.crossref.org/works?query="+q+"&rows="+fmt.Sprint(limit(req.Limit)), nil, &raw); err != nil {
			return nil, err
		}
		var out []capability.ScholarWork
		for _, r := range raw.Message.Items {
			title := ""
			if len(r.Title) > 0 {
				title = r.Title[0]
			}
			out = append(out, capability.ScholarWork{DOI: r.DOI, Title: title, URL: r.URL, Provider: a.id})
		}
		return out, nil
	case "arxiv":
		txt, err := a.getText(ctx, "https://export.arxiv.org/api/query?search_query=all:"+q+"&start=0&max_results="+fmt.Sprint(limit(req.Limit)), nil)
		if err != nil {
			return nil, err
		}
		return parseArxiv(txt, a.id), nil
	case "pubmed":
		key, _ := a.creds.Get("pubmed", "NCBI_API_KEY")
		var search struct {
			ESearchResult struct {
				IDList []string `json:"idlist"`
			} `json:"esearchresult"`
		}
		u := "https://eutils.ncbi.nlm.nih.gov/entrez/eutils/esearch.fcgi?db=pubmed&retmode=json&term=" + q + "&retmax=" + fmt.Sprint(limit(req.Limit))
		if err := a.getJSONWithOptionalAPIKey(ctx, u, "api_key", key.Value, &search); err != nil {
			return nil, err
		}
		if len(search.ESearchResult.IDList) == 0 {
			return nil, nil
		}
		sumURL := "https://eutils.ncbi.nlm.nih.gov/entrez/eutils/esummary.fcgi?db=pubmed&retmode=json&id=" + strings.Join(search.ESearchResult.IDList, ",")
		var sum struct {
			Result map[string]json.RawMessage `json:"result"`
		}
		if err := a.getJSONWithOptionalAPIKey(ctx, sumURL, "api_key", key.Value, &sum); err != nil {
			return nil, err
		}
		var out []capability.ScholarWork
		for _, id := range search.ESearchResult.IDList {
			var r struct {
				UID, Title, FullJournalName string
				PubDate                     string
			}
			if err := json.Unmarshal(sum.Result[id], &r); err != nil {
				continue
			}
			out = append(out, capability.ScholarWork{ID: id, Title: r.Title, URL: "https://pubmed.ncbi.nlm.nih.gov/" + id + "/", Provider: a.id})
		}
		return out, nil
	case "semantic_scholar":
		var raw struct {
			Data []semanticScholarPaper `json:"data"`
		}
		u := "https://api.semanticscholar.org/graph/v1/paper/search?query=" + q + "&limit=" + fmt.Sprint(limit(req.Limit)) + "&fields=paperId,title,abstract,url,year,externalIds,citationCount,referenceCount,authors,venue"
		if err := a.getJSON(ctx, u, nil, &raw); err != nil {
			return nil, err
		}
		var out []capability.ScholarWork
		for _, r := range raw.Data {
			w := capability.ScholarWork{ID: r.PaperID, DOI: r.ExternalIDs.DOI, Title: r.Title, Abstract: r.Abstract, URL: r.URL, Year: r.Year, Authors: semanticAuthors(r.Authors), Venue: r.Venue, CitationCount: r.CitationCount, ReferenceCount: r.ReferenceCount, Provider: a.id}
			if req.IncludeRaw {
				w.Raw = r
			}
			out = append(out, w)
		}
		return out, nil
	case "datacite":
		var raw struct {
			Data []struct {
				ID         string
				Attributes struct {
					DOI             string
					Titles          []struct{ Title string }
					URL             string
					PublicationYear int `json:"publicationYear"`
				}
			} `json:"data"`
		}
		if err := a.getJSON(ctx, "https://api.datacite.org/dois?query="+q+"&page[size]="+fmt.Sprint(limit(req.Limit)), nil, &raw); err != nil {
			return nil, err
		}
		var out []capability.ScholarWork
		for _, r := range raw.Data {
			title := ""
			if len(r.Attributes.Titles) > 0 {
				title = r.Attributes.Titles[0].Title
			}
			out = append(out, capability.ScholarWork{ID: r.ID, DOI: r.Attributes.DOI, Title: title, URL: r.Attributes.URL, Year: r.Attributes.PublicationYear, Provider: a.id})
		}
		return out, nil
	case "europepmc":
		var raw struct {
			ResultList struct {
				Result []struct {
					ID, DOI, Title, AbstractText string
					Year                         string
				} `json:"result"`
			} `json:"resultList"`
		}
		if err := a.getJSON(ctx, "https://www.ebi.ac.uk/europepmc/webservices/rest/search?format=json&query="+q+"&pageSize="+fmt.Sprint(limit(req.Limit)), nil, &raw); err != nil {
			return nil, err
		}
		var out []capability.ScholarWork
		for _, r := range raw.ResultList.Result {
			out = append(out, capability.ScholarWork{ID: r.ID, DOI: r.DOI, Title: r.Title, Abstract: r.AbstractText, URL: "https://europepmc.org/article/MED/" + r.ID, Provider: a.id})
		}
		return out, nil
	case "doaj":
		var raw struct {
			Results []struct {
				BibJSON struct {
					Title      string
					Identifier []struct{ Type, ID string }
					Link       []struct{ URL string }
				} `json:"bibjson"`
			} `json:"results"`
		}
		if err := a.getJSON(ctx, "https://doaj.org/api/search/articles/"+q+"?pageSize="+fmt.Sprint(limit(req.Limit)), nil, &raw); err != nil {
			return nil, err
		}
		var out []capability.ScholarWork
		for _, r := range raw.Results {
			doi := ""
			for _, id := range r.BibJSON.Identifier {
				if strings.EqualFold(id.Type, "doi") {
					doi = id.ID
				}
			}
			link := ""
			if len(r.BibJSON.Link) > 0 {
				link = r.BibJSON.Link[0].URL
			}
			out = append(out, capability.ScholarWork{DOI: doi, Title: r.BibJSON.Title, URL: link, Provider: a.id})
		}
		return out, nil
	default:
		return nil, ProviderError{Code: "unsupported_capability", Message: "scholar search unsupported"}
	}
}

type semanticScholarPaper struct {
	PaperID        string `json:"paperId"`
	Title          string
	Abstract       string
	URL            string
	Year           int
	Venue          string
	CitationCount  int `json:"citationCount"`
	ReferenceCount int `json:"referenceCount"`
	ExternalIDs    struct {
		DOI string
	} `json:"externalIds"`
	Authors []struct {
		Name string
	} `json:"authors"`
}

func normalizeSemanticScholarPaper(raw semanticScholarPaper, provider string, includeRaw bool) *capability.ScholarlyRecord {
	rec := &capability.ScholarlyRecord{
		ID:             raw.PaperID,
		DOI:            raw.ExternalIDs.DOI,
		Title:          raw.Title,
		Abstract:       raw.Abstract,
		Year:           raw.Year,
		URL:            raw.URL,
		Authors:        semanticAuthors(raw.Authors),
		Venue:          raw.Venue,
		CitationCount:  raw.CitationCount,
		ReferenceCount: raw.ReferenceCount,
		Provider:       provider,
	}
	if includeRaw {
		rec.Raw = raw
	}
	return rec
}

func citationSummary(provider string, returned, total int, note string) map[string]any {
	s := map[string]any{
		"provider":         provider,
		"records_returned": returned,
		"count":            returned,
		"source_scope":     citationScope(provider),
		"coverage_note":    note,
	}
	if total > 0 {
		s["provider_total"] = total
	}
	return s
}

func citationScope(provider string) string {
	switch provider {
	case "opencitations":
		return "indexed DOI-to-DOI citation links"
	case "openalex":
		return "OpenAlex works citing the requested work"
	case "semantic_scholar":
		return "Semantic Scholar graph citing papers"
	case "crossref":
		return "fallback bibliographic metadata only"
	default:
		return "provider-specific citation coverage"
	}
}

func semanticAuthors(in []struct{ Name string }) []string {
	var out []string
	for _, a := range in {
		if strings.TrimSpace(a.Name) != "" {
			out = append(out, a.Name)
		}
	}
	return out
}

func normalizeOpenAlexWork(raw map[string]any, provider string, includeRaw bool) *capability.ScholarlyRecord {
	rec := &capability.ScholarlyRecord{
		ID:             stringField(raw, "id"),
		DOI:            stringField(raw, "doi"),
		Title:          stringField(raw, "title", "display_name"),
		Year:           intField(raw, "publication_year"),
		URL:            nestedString(raw, "primary_location", "landing_page_url"),
		CitationCount:  intField(raw, "cited_by_count"),
		ReferenceCount: len(anySlice(raw["referenced_works"])),
		Provider:       provider,
	}
	if rec.URL == "" {
		rec.URL = nestedString(raw, "best_oa_location", "landing_page_url")
	}
	if includeRaw {
		rec.Raw = raw
	}
	return rec
}

func normalizeOpenAlexOA(raw map[string]any, provider string, includeRaw bool) *capability.OpenAccessSummary {
	loc, _ := raw["best_oa_location"].(map[string]any)
	if loc == nil {
		loc, _ = raw["primary_location"].(map[string]any)
	}
	oa := &capability.OpenAccessSummary{
		IsOA:     nestedBool(raw, "open_access", "is_oa") || boolField(raw, "is_oa"),
		Status:   nestedString(raw, "open_access", "oa_status"),
		URL:      stringField(loc, "landing_page_url", "url"),
		PDFURL:   stringField(loc, "pdf_url", "url_for_pdf"),
		License:  stringField(loc, "license"),
		HostType: stringField(loc, "source_type"),
		Provider: provider,
	}
	if !oa.IsOA && oa.Status == "" && oa.URL == "" && oa.PDFURL == "" {
		return nil
	}
	if includeRaw {
		oa.Raw = raw
	}
	return oa
}

func normalizeCrossrefWork(raw map[string]any, provider string, includeRaw bool) *capability.ScholarlyRecord {
	msg, _ := raw["message"].(map[string]any)
	rec := &capability.ScholarlyRecord{
		DOI:           stringField(msg, "DOI", "doi"),
		Title:         firstString(msg["title"]),
		URL:           stringField(msg, "URL", "url"),
		Venue:         firstString(msg["container-title"]),
		CitationCount: intField(msg, "is-referenced-by-count"),
		Provider:      provider,
	}
	rec.Year = crossrefYear(msg)
	rec.Authors = crossrefAuthors(msg)
	if includeRaw {
		rec.Raw = raw
	}
	return rec
}

func normalizeUnpaywall(raw map[string]any, provider string, includeRaw bool) *capability.OpenAccessSummary {
	loc, _ := raw["best_oa_location"].(map[string]any)
	oa := &capability.OpenAccessSummary{
		IsOA:        boolField(raw, "is_oa"),
		Status:      stringField(raw, "oa_status"),
		URL:         stringField(loc, "url", "url_for_landing_page"),
		PDFURL:      stringField(loc, "url_for_pdf"),
		License:     stringField(loc, "license"),
		HostType:    stringField(loc, "host_type"),
		JournalName: stringField(raw, "journal_name"),
		Publisher:   stringField(raw, "publisher"),
		Provider:    provider,
	}
	if includeRaw {
		oa.Raw = raw
	}
	return oa
}

func includeRaw(include bool, raw any) any {
	if include {
		return raw
	}
	return nil
}

func looksLikeDOI(v string) bool {
	v = strings.TrimSpace(strings.ToLower(v))
	return strings.HasPrefix(v, "10.") || strings.HasPrefix(v, "doi:10.")
}

func cleanDOI(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(strings.TrimPrefix(v, "doi:"), "DOI:")
	return v
}

func stringField(m map[string]any, names ...string) string {
	if m == nil {
		return ""
	}
	for _, name := range names {
		if v, ok := m[name]; ok {
			switch x := v.(type) {
			case string:
				return strings.TrimSpace(x)
			case fmt.Stringer:
				return strings.TrimSpace(x.String())
			}
		}
	}
	return ""
}

func nestedString(m map[string]any, parent, child string) string {
	n, _ := m[parent].(map[string]any)
	return stringField(n, child)
}

func intField(m map[string]any, names ...string) int {
	for _, name := range names {
		switch v := m[name].(type) {
		case int:
			return v
		case int64:
			return int(v)
		case float64:
			return int(v)
		case json.Number:
			n, _ := v.Int64()
			return int(n)
		}
	}
	return 0
}

func boolField(m map[string]any, name string) bool {
	v, _ := m[name].(bool)
	return v
}

func nestedBool(m map[string]any, parent, child string) bool {
	n, _ := m[parent].(map[string]any)
	return boolField(n, child)
}

func firstString(v any) string {
	switch x := v.(type) {
	case []any:
		for _, item := range x {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				return s
			}
		}
	case []string:
		if len(x) > 0 {
			return x[0]
		}
	case string:
		return x
	}
	return ""
}

func anySlice(v any) []any {
	x, _ := v.([]any)
	return x
}

func crossrefYear(msg map[string]any) int {
	for _, key := range []string{"published-print", "published-online", "published"} {
		pub, _ := msg[key].(map[string]any)
		parts, _ := pub["date-parts"].([]any)
		if len(parts) == 0 {
			continue
		}
		first, _ := parts[0].([]any)
		if len(first) == 0 {
			continue
		}
		if y, ok := first[0].(float64); ok {
			return int(y)
		}
	}
	return 0
}

func crossrefAuthors(msg map[string]any) []string {
	authors, _ := msg["author"].([]any)
	var out []string
	for _, item := range authors {
		a, _ := item.(map[string]any)
		name := strings.TrimSpace(strings.TrimSpace(stringField(a, "given") + " " + stringField(a, "family")))
		if name == "" {
			name = stringField(a, "name")
		}
		if name != "" {
			out = append(out, name)
		}
	}
	return out
}

func yearFromDate(v string) int {
	if len(v) >= 4 {
		var y int
		if _, err := fmt.Sscanf(v[:4], "%d", &y); err == nil {
			return y
		}
	}
	return 0
}

func (a HTTPAdapter) directFetch(ctx context.Context, target string) (capability.ExtractedDocument, error) {
	txt, contentType, err := a.getTextWithMetadata(ctx, target, map[string]string{"User-Agent": "forage/0.1"})
	if err != nil {
		return capability.ExtractedDocument{}, err
	}
	if contentType != "" && !strings.Contains(strings.ToLower(contentType), "text/html") && !strings.Contains(strings.ToLower(contentType), "text/plain") {
		return capability.ExtractedDocument{}, ProviderError{Code: "unsupported_content_type", Message: "direct fetch does not extract " + contentType}
	}
	title := extractTitle(txt)
	plain := stripHTML(txt)
	return capability.ExtractedDocument{URL: target, Title: title, RetrievedAt: time.Now().UTC(), HTML: txt, PlainText: plain, Markdown: plain, Provider: a.id, ExtractionMethod: "direct_http", QualityScore: quality(plain), ContentHash: hash(plain)}, nil
}

func (a HTTPAdapter) jinaFetch(ctx context.Context, target string) (capability.ExtractedDocument, error) {
	key, _ := a.creds.Get("jina", "JINA_API_KEY")
	u := "https://r.jina.ai/" + target
	md, err := a.getText(ctx, u, bearerHeaders(key.Value))
	if err != nil {
		return capability.ExtractedDocument{}, err
	}
	return capability.ExtractedDocument{URL: target, RetrievedAt: time.Now().UTC(), Markdown: md, PlainText: stripMarkdown(md), Provider: a.id, ExtractionMethod: "jina_reader", QualityScore: quality(md), ContentHash: hash(md)}, nil
}

func (a HTTPAdapter) browserbaseFetch(ctx context.Context, target string) (capability.ExtractedDocument, error) {
	key, _ := a.creds.Get("browserbase", "BROWSERBASE_API_KEY")
	body := map[string]any{"url": target, "format": "markdown", "allowRedirects": true}
	var raw struct {
		StatusCode  int               `json:"statusCode"`
		Headers     map[string]string `json:"headers"`
		Content     string            `json:"content"`
		ContentType string            `json:"contentType"`
		Encoding    string            `json:"encoding"`
	}
	if err := a.postJSON(ctx, "https://api.browserbase.com/v1/fetch", map[string]string{"X-BB-API-Key": key.Value}, body, &raw); err != nil {
		return capability.ExtractedDocument{}, err
	}
	md := raw.Content
	return capability.ExtractedDocument{URL: target, RetrievedAt: time.Now().UTC(), Markdown: md, PlainText: stripMarkdown(md), Provider: a.id, ExtractionMethod: "browserbase_fetch_markdown", QualityScore: quality(md), ContentHash: hash(md), Raw: map[string]any{"status_code": raw.StatusCode, "content_type": raw.ContentType, "encoding": raw.Encoding, "headers": raw.Headers}}, nil
}

func (a HTTPAdapter) firecrawlExtract(ctx context.Context, target string) (capability.ExtractedDocument, error) {
	key, _ := a.creds.Get("firecrawl", "FIRECRAWL_API_KEY")
	body := map[string]any{"url": target, "formats": []string{"markdown", "html"}}
	var raw struct {
		Success bool
		Data    struct {
			Markdown, HTML string
			Metadata       map[string]any
		}
	}
	if err := a.postJSON(ctx, "https://api.firecrawl.dev/v1/scrape", map[string]string{"Authorization": "Bearer " + key.Value}, body, &raw); err != nil {
		return capability.ExtractedDocument{}, err
	}
	return capability.ExtractedDocument{URL: target, RetrievedAt: time.Now().UTC(), Markdown: raw.Data.Markdown, HTML: raw.Data.HTML, PlainText: stripMarkdown(raw.Data.Markdown), Provider: a.id, ExtractionMethod: "firecrawl_scrape", QualityScore: quality(raw.Data.Markdown), ContentHash: hash(raw.Data.Markdown), Raw: raw.Data.Metadata}, nil
}

func (a HTTPAdapter) scrapingAntExtract(ctx context.Context, target string) (capability.ExtractedDocument, error) {
	key, _ := a.creds.Get("scrapingant", "SCRAPINGANT_API_KEY")
	u := "https://api.scrapingant.com/v2/general?x-api-key=" + url.QueryEscape(key.Value) + "&url=" + url.QueryEscape(target) + "&return_text=true"
	txt, err := a.getText(ctx, u, nil)
	if err != nil {
		return capability.ExtractedDocument{}, err
	}
	plain := txt
	if strings.Contains(strings.ToLower(txt), "<html") || strings.Contains(strings.ToLower(txt), "<body") {
		plain = stripHTML(txt)
	}
	return capability.ExtractedDocument{URL: target, Title: extractTitle(txt), RetrievedAt: time.Now().UTC(), HTML: htmlIfPresent(txt), PlainText: plain, Markdown: plain, Provider: a.id, ExtractionMethod: "scrapingant_text", QualityScore: quality(plain), ContentHash: hash(plain)}, nil
}

func (a HTTPAdapter) getJSON(ctx context.Context, u string, headers map[string]string, dest any) error {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return providerHTTPError(resp)
	}
	if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
		return ProviderError{Code: "malformed_response", Message: "provider returned malformed JSON"}
	}
	return nil
}

func (a HTTPAdapter) getJSONWithOptionalAPIKey(ctx context.Context, u, keyParam, keyValue string, dest any) error {
	keyValue = strings.TrimSpace(keyValue)
	if keyValue == "" {
		return a.getJSON(ctx, u, nil, dest)
	}
	sep := "?"
	if strings.Contains(u, "?") {
		sep = "&"
	}
	err := a.getJSON(ctx, u+sep+url.QueryEscape(keyParam)+"="+url.QueryEscape(keyValue), nil, dest)
	if err == nil {
		return nil
	}
	return a.getJSON(ctx, u, nil, dest)
}

func (a HTTPAdapter) postJSON(ctx context.Context, u string, headers map[string]string, body any, dest any) error {
	b, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return providerHTTPError(resp)
	}
	if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
		return ProviderError{Code: "malformed_response", Message: "provider returned malformed JSON"}
	}
	return nil
}

func (a HTTPAdapter) getText(ctx context.Context, u string, headers map[string]string) (string, error) {
	txt, _, err := a.getTextWithMetadata(ctx, u, headers)
	return txt, err
}

func (a HTTPAdapter) getTextWithMetadata(ctx context.Context, u string, headers map[string]string) (string, string, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", providerHTTPError(resp)
	}
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	return string(b), resp.Header.Get("Content-Type"), nil
}

func providerHTTPError(resp *http.Response) error {
	code := "provider_http_error"
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		code = "auth_failed"
	}
	if resp.StatusCode == 429 {
		code = "rate_limited"
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	observed := observedQuotaHeaders(resp)
	if len(body) > 0 {
		observed = strings.TrimSpace(strings.Join([]string{observed, "body=" + string(body)}, "; "))
	}
	return ProviderError{Code: code, Message: fmt.Sprintf("provider returned HTTP %d", resp.StatusCode), RetryAfter: firstHeaderValue(resp.Header.Get("Retry-After")), ResetAt: firstHeader(resp, "x-ratelimit-reset", "ratelimit-reset", "x-rate-limit-reset"), Observed: observed, HTTPStatus: resp.StatusCode}
}

func result(provider string, rank int, u, title, snippet, typ string) capability.SearchResult {
	canon := canonicalURL("", u)
	return capability.SearchResult{URL: u, CanonicalURL: canon, Title: cleanText(title, 300), Snippet: cleanText(stripHTML(snippet), 700), Provider: provider, ProviderRank: rank + 1, ResultType: typ, SourceDomain: domain(canon)}
}
func limit(n int) int {
	if n <= 0 {
		return 10
	}
	if n > 100 {
		return 100
	}
	return n
}
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func domain(u string) string {
	parsed, err := url.Parse(u)
	if err != nil {
		return ""
	}
	return parsed.Hostname()
}
func stripHTML(s string) string {
	for _, tag := range []string{"script", "style", "noscript"} {
		s = regexp.MustCompile(`(?is)<`+tag+`[^>]*>.*?</`+tag+`>`).ReplaceAllString(s, " ")
	}
	re := regexp.MustCompile(`<[^>]+>`)
	s = re.ReplaceAllString(s, " ")
	s = html.UnescapeString(s)
	return strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(s, " "))
}
func stripMarkdown(s string) string { return strings.TrimSpace(strings.ReplaceAll(s, "#", "")) }
func cleanText(s string, max int) string {
	s = strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(s, " "))
	if max > 0 && len(s) > max {
		return strings.TrimSpace(s[:max]) + "..."
	}
	return s
}
func extractTitle(s string) string {
	re := regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	m := re.FindStringSubmatch(s)
	if len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

func htmlIfPresent(s string) string {
	lower := strings.ToLower(s)
	if strings.Contains(lower, "<html") || strings.Contains(lower, "<body") || strings.Contains(lower, "<!doctype") {
		return s
	}
	return ""
}
func quality(s string) float64 {
	l := len(strings.TrimSpace(s))
	if l > 2000 {
		return .95
	}
	if l > 500 {
		return .75
	}
	if l > 100 {
		return .45
	}
	return .1
}
func hash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func firstHeader(resp *http.Response, names ...string) string {
	for _, name := range names {
		if v := resp.Header.Get(name); v != "" {
			return firstHeaderValue(v)
		}
	}
	return ""
}

func observedQuotaHeaders(resp *http.Response) string {
	var parts []string
	for _, h := range []string{"x-ratelimit-limit", "x-ratelimit-remaining", "x-ratelimit-reset", "x-ratelimit-used", "x-ratelimit-credits-used", "ratelimit-limit", "ratelimit-remaining", "ratelimit-reset", "ratelimit-used", "retry-after"} {
		if v := resp.Header.Get(h); v != "" {
			parts = append(parts, h+"="+v)
		}
	}
	return strings.Join(parts, "; ")
}

func firstHeaderValue(v string) string {
	parts := strings.Split(v, ",")
	if len(parts) == 0 {
		return strings.TrimSpace(v)
	}
	return strings.TrimSpace(parts[0])
}

func bearerHeaders(token string) map[string]string {
	if strings.TrimSpace(token) == "" {
		return nil
	}
	return map[string]string{"Authorization": "Bearer " + strings.TrimSpace(token)}
}

func parseMarkdownLinks(provider, txt, typ string) []capability.SearchResult {
	var out []capability.SearchResult
	re := regexp.MustCompile(`(?m)^\s*\d+\.\s*\[(.*?)\]\((https?://[^)]+)\)`)
	for i, m := range re.FindAllStringSubmatch(txt, -1) {
		out = append(out, result(provider, i, m[2], m[1], "", typ))
	}
	return out
}

func parseArxiv(txt, provider string) []capability.ScholarWork {
	var out []capability.ScholarWork
	entryRe := regexp.MustCompile(`(?s)<entry>(.*?)</entry>`)
	titleRe := regexp.MustCompile(`(?s)<title>(.*?)</title>`)
	idRe := regexp.MustCompile(`(?s)<id>(.*?)</id>`)
	for _, e := range entryRe.FindAllStringSubmatch(txt, -1) {
		title := ""
		id := ""
		if m := titleRe.FindStringSubmatch(e[1]); len(m) > 1 {
			title = strings.TrimSpace(m[1])
		}
		if m := idRe.FindStringSubmatch(e[1]); len(m) > 1 {
			id = strings.TrimSpace(m[1])
		}
		out = append(out, capability.ScholarWork{ID: id, Title: title, URL: id, Provider: provider})
	}
	return out
}

func (a HTTPAdapter) localCrawl(ctx context.Context, start string, maxPages int) ([]capability.ExtractedDocument, error) {
	base, err := url.Parse(start)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	queue := []string{start}
	var docs []capability.ExtractedDocument
	for len(queue) > 0 && len(docs) < maxPages {
		u := queue[0]
		queue = queue[1:]
		if seen[u] {
			continue
		}
		seen[u] = true
		doc, err := a.directFetch(ctx, u)
		if err != nil {
			continue
		}
		docs = append(docs, doc)
		for _, link := range extractLinks(doc.HTML, u) {
			parsed, err := url.Parse(link)
			if err != nil || parsed.Hostname() != base.Hostname() || seen[link] {
				continue
			}
			queue = append(queue, link)
		}
	}
	if len(docs) == 0 {
		return nil, ProviderError{Code: "not_found", Message: "crawl returned no documents"}
	}
	return docs, nil
}

func extractLinks(htmlText string, baseURL string) []string {
	base, _ := url.Parse(baseURL)
	re := regexp.MustCompile(`(?i)href=["']([^"'#]+)["']`)
	seen := map[string]bool{}
	var out []string
	for _, m := range re.FindAllStringSubmatch(htmlText, -1) {
		ref, err := url.Parse(strings.TrimSpace(m[1]))
		if err != nil {
			continue
		}
		abs := base.ResolveReference(ref)
		abs.Fragment = ""
		u := abs.String()
		if strings.HasPrefix(u, "http") && !seen[u] {
			seen[u] = true
			out = append(out, u)
		}
	}
	return out
}
