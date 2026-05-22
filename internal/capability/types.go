package capability

import "time"

const (
	SearchWeb      = "search.web"
	SearchNews     = "search.news"
	SearchScholar  = "search.scholar"
	SearchPlatform = "search.platform"
	FetchURL       = "fetch.url"
	ExtractArticle = "extract.article"
	ArchiveLookup  = "archive.lookup"
	EnrichDOI      = "enrich.doi"
	EnrichPaper    = "enrich.paper"
	EnrichAuthor   = "enrich.author"
	CitationsDOI   = "citations.doi"
	CorpusQuery    = "corpus.query"
	RenderBrowser  = "render.browser"
	CrawlSite      = "crawl.site"
)

type SearchRequest struct {
	Query            string   `json:"query"`
	Capability       string   `json:"capability"`
	Limit            int      `json:"limit"`
	Freshness        string   `json:"freshness,omitempty"`
	Site             string   `json:"site,omitempty"`
	Providers        []string `json:"providers,omitempty"`
	ExcludeProviders []string `json:"exclude_providers,omitempty"`
	CacheMode        string   `json:"cache_mode"`
	ExplainRouting   bool     `json:"explain_routing"`
}

type FetchRequest struct {
	URL              string   `json:"url"`
	Providers        []string `json:"providers,omitempty"`
	ExcludeProviders []string `json:"exclude_providers,omitempty"`
	CacheMode        string   `json:"cache_mode"`
	ExplainRouting   bool     `json:"explain_routing"`
}

type ExtractRequest struct {
	URL              string   `json:"url"`
	Providers        []string `json:"providers,omitempty"`
	ExcludeProviders []string `json:"exclude_providers,omitempty"`
	CacheMode        string   `json:"cache_mode"`
	ExplainRouting   bool     `json:"explain_routing"`
}

type DataRequest struct {
	Capability       string         `json:"capability"`
	Query            string         `json:"query,omitempty"`
	URL              string         `json:"url,omitempty"`
	ID               string         `json:"id,omitempty"`
	Providers        []string       `json:"providers,omitempty"`
	ExcludeProviders []string       `json:"exclude_providers,omitempty"`
	CacheMode        string         `json:"cache_mode"`
	ExplainRouting   bool           `json:"explain_routing"`
	Limit            int            `json:"limit,omitempty"`
	MaxPages         int            `json:"max_pages,omitempty"`
	Options          map[string]any `json:"options,omitempty"`
}

type DataResponse struct {
	Capability  string              `json:"capability"`
	Data        any                 `json:"data"`
	Routing     *RoutingDiagnostics `json:"routing,omitempty"`
	CacheStatus string              `json:"cache_status,omitempty"`
}

type SearchResponse struct {
	Query       string              `json:"query"`
	Capability  string              `json:"capability"`
	Results     []SearchResult      `json:"results"`
	Routing     *RoutingDiagnostics `json:"routing,omitempty"`
	CacheStatus string              `json:"cache_status,omitempty"`
}

type FetchResponse struct {
	Document    ExtractedDocument   `json:"document"`
	Routing     *RoutingDiagnostics `json:"routing,omitempty"`
	CacheStatus string              `json:"cache_status,omitempty"`
}

type SearchResult struct {
	URL          string     `json:"url,omitempty"`
	CanonicalURL string     `json:"canonical_url,omitempty"`
	Title        string     `json:"title,omitempty"`
	Snippet      string     `json:"snippet,omitempty"`
	PublishedAt  *time.Time `json:"published_at,omitempty"`
	SourceName   string     `json:"source_name,omitempty"`
	SourceDomain string     `json:"source_domain,omitempty"`
	Provider     string     `json:"provider"`
	ProviderRank int        `json:"provider_rank"`
	Score        float64    `json:"score,omitempty"`
	ResultType   string     `json:"result_type"`
	Language     string     `json:"language,omitempty"`
	Raw          any        `json:"raw,omitempty"`
}

type ExtractedDocument struct {
	URL              string     `json:"url"`
	CanonicalURL     string     `json:"canonical_url,omitempty"`
	Title            string     `json:"title,omitempty"`
	Author           string     `json:"author,omitempty"`
	PublishedAt      *time.Time `json:"published_at,omitempty"`
	RetrievedAt      time.Time  `json:"retrieved_at"`
	Markdown         string     `json:"markdown,omitempty"`
	PlainText        string     `json:"plain_text,omitempty"`
	HTML             string     `json:"html,omitempty"`
	Provider         string     `json:"provider"`
	ExtractionMethod string     `json:"extraction_method"`
	ContentHash      string     `json:"content_hash,omitempty"`
	QualityScore     float64    `json:"quality_score,omitempty"`
	Raw              any        `json:"raw,omitempty"`
}

type ScholarWork struct {
	ID       string `json:"id,omitempty"`
	DOI      string `json:"doi,omitempty"`
	Title    string `json:"title,omitempty"`
	Abstract string `json:"abstract,omitempty"`
	Year     int    `json:"year,omitempty"`
	URL      string `json:"url,omitempty"`
	Provider string `json:"provider"`
}

type NewsResult = SearchResult
type PlatformResult = SearchResult

type ArchiveRecord struct {
	URL        string `json:"url"`
	ArchiveURL string `json:"archive_url,omitempty"`
	Timestamp  string `json:"timestamp,omitempty"`
	Status     string `json:"status,omitempty"`
	MimeType   string `json:"mime_type,omitempty"`
	Provider   string `json:"provider"`
}

type ProviderAttempt struct {
	Provider   string `json:"provider"`
	Status     string `json:"status"`
	Reason     string `json:"reason,omitempty"`
	StartedAt  string `json:"started_at,omitempty"`
	FinishedAt string `json:"finished_at,omitempty"`
}

type RoutingDiagnostics struct {
	Capability    string            `json:"capability"`
	ProvidersUsed []string          `json:"providers_used,omitempty"`
	Attempts      []ProviderAttempt `json:"attempts,omitempty"`
	Skipped       []ProviderAttempt `json:"skipped,omitempty"`
}

type CapabilityError struct {
	Code       string            `json:"code"`
	Capability string            `json:"capability"`
	Message    string            `json:"message"`
	Attempts   []ProviderAttempt `json:"attempts,omitempty"`
}
