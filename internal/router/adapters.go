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
		"brave", "jina", "browserbase", "tavily", "exa", "serpapi", "serpstack", "google_cse",
		"direct", "firecrawl", "scrapingant", "apify",
		"gdelt", "guardian", "currents", "gnews", "newsapi", "mediastack", "worldnews",
		"hackernews", "forem", "reddit",
		"openalex", "semantic_scholar", "crossref", "arxiv", "pubmed", "datacite", "europepmc", "doaj",
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
			out = append(out, capability.SearchResult{URL: w.URL, Title: w.Title, Snippet: w.Abstract, Provider: w.Provider, ProviderRank: i + 1, ResultType: "paper"})
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

func (a HTTPAdapter) searchWeb(ctx context.Context, req capability.SearchRequest) ([]capability.SearchResult, error) {
	q := req.Query
	if req.Site != "" {
		q += " site:" + req.Site
	}
	switch a.id {
	case "brave":
		key, _ := a.creds.Get("brave", "BRAVE_API_KEY")
		u := "https://api.search.brave.com/res/v1/web/search?q=" + url.QueryEscape(q) + "&count=" + fmt.Sprint(limit(req.Limit))
		h := map[string]string{"X-Subscription-Token": key.Value, "Accept": "application/json"}
		var raw struct {
			Web struct {
				Results []struct{ Title, URL, Description, Profile string } `json:"results"`
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
		u := "https://s.jina.ai/" + url.QueryEscape(q)
		var raw []struct{ Title, URL, Content string }
		err := a.getJSON(ctx, u, nil, &raw)
		if err != nil {
			txt, terr := a.getText(ctx, u, nil)
			if terr != nil {
				return nil, err
			}
			return parseMarkdownLinks(a.id, txt, "web"), nil
		}
		var out []capability.SearchResult
		for i, r := range raw {
			out = append(out, result(a.id, i, r.URL, r.Title, r.Content, "web"))
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
	case "google_cse":
		key, _ := a.creds.Get("google_cse", "GOOGLE_CSE_API_KEY")
		cx, _ := a.creds.GetField("google_cse", "cx", "GOOGLE_CSE_CX")
		if cx.Value == "" {
			return nil, ProviderError{Code: "auth_missing", Message: "GOOGLE_CSE_CX missing"}
		}
		u := "https://www.googleapis.com/customsearch/v1?key=" + url.QueryEscape(key.Value) + "&cx=" + url.QueryEscape(cx.Value) + "&q=" + url.QueryEscape(q)
		var raw struct {
			Items []struct{ Title, Link, Snippet string } `json:"items"`
		}
		if err := a.getJSON(ctx, u, nil, &raw); err != nil {
			return nil, err
		}
		var out []capability.SearchResult
		for i, r := range raw.Items {
			out = append(out, result(a.id, i, r.Link, r.Title, r.Snippet, "web"))
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
		u := "https://eutils.ncbi.nlm.nih.gov/entrez/eutils/esearch.fcgi?db=pubmed&retmode=json&term=" + q + "&retmax=" + fmt.Sprint(limit(req.Limit))
		if key.Value != "" {
			u += "&api_key=" + url.QueryEscape(key.Value)
		}
		var search struct {
			ESearchResult struct {
				IDList []string `json:"idlist"`
			} `json:"esearchresult"`
		}
		if err := a.getJSON(ctx, u, nil, &search); err != nil {
			return nil, err
		}
		if len(search.ESearchResult.IDList) == 0 {
			return nil, nil
		}
		sumURL := "https://eutils.ncbi.nlm.nih.gov/entrez/eutils/esummary.fcgi?db=pubmed&retmode=json&id=" + strings.Join(search.ESearchResult.IDList, ",")
		if key.Value != "" {
			sumURL += "&api_key=" + url.QueryEscape(key.Value)
		}
		var sum struct {
			Result map[string]struct {
				UID, Title, FullJournalName string
				PubDate                     string
			} `json:"result"`
		}
		if err := a.getJSON(ctx, sumURL, nil, &sum); err != nil {
			return nil, err
		}
		var out []capability.ScholarWork
		for _, id := range search.ESearchResult.IDList {
			r := sum.Result[id]
			out = append(out, capability.ScholarWork{ID: id, Title: r.Title, URL: "https://pubmed.ncbi.nlm.nih.gov/" + id + "/", Provider: a.id})
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
	u := "https://r.jina.ai/" + target
	md, err := a.getText(ctx, u, nil)
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
	return capability.ExtractedDocument{URL: target, RetrievedAt: time.Now().UTC(), PlainText: txt, Markdown: txt, Provider: a.id, ExtractionMethod: "scrapingant_text", QualityScore: quality(txt), ContentHash: hash(txt)}, nil
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
	return json.NewDecoder(resp.Body).Decode(dest)
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
	return json.NewDecoder(resp.Body).Decode(dest)
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
	return ProviderError{Code: code, Message: fmt.Sprintf("provider returned HTTP %d", resp.StatusCode), RetryAfter: resp.Header.Get("Retry-After"), ResetAt: firstHeader(resp, "x-ratelimit-reset", "ratelimit-reset", "x-rate-limit-reset"), Observed: observed, HTTPStatus: resp.StatusCode}
}

func result(provider string, rank int, u, title, snippet, typ string) capability.SearchResult {
	return capability.SearchResult{URL: u, CanonicalURL: u, Title: title, Snippet: stripHTML(snippet), Provider: provider, ProviderRank: rank + 1, ResultType: typ, SourceDomain: domain(u)}
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
func extractTitle(s string) string {
	re := regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	m := re.FindStringSubmatch(s)
	if len(m) > 1 {
		return strings.TrimSpace(m[1])
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
			return v
		}
	}
	return ""
}

func observedQuotaHeaders(resp *http.Response) string {
	var parts []string
	for _, h := range []string{"x-ratelimit-limit", "x-ratelimit-remaining", "x-ratelimit-reset", "ratelimit-limit", "ratelimit-remaining", "ratelimit-reset", "retry-after"} {
		if v := resp.Header.Get(h); v != "" {
			parts = append(parts, h+"="+v)
		}
	}
	return strings.Join(parts, "; ")
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
