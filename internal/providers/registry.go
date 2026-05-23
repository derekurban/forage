package providers

import "sort"

type AuthType string
type ImplementationStatus string

const (
	AuthNone   AuthType = "none"
	AuthAPIKey AuthType = "api_key"
	AuthOAuth  AuthType = "oauth"
	AuthCustom AuthType = "custom"

	MetadataOnly        ImplementationStatus = "metadata_only"
	DoctorSupported     ImplementationStatus = "doctor_supported"
	CapabilitySupported ImplementationStatus = "capability_supported"
	LegacyOptional      ImplementationStatus = "legacy_optional"
)

type Provider struct {
	ID               string               `json:"id"`
	Name             string               `json:"name"`
	Category         string               `json:"category"`
	Capabilities     []string             `json:"capabilities"`
	AuthType         AuthType             `json:"auth_type"`
	OptionalAuth     bool                 `json:"optional_auth,omitempty"`
	EnvVar           string               `json:"env_var,omitempty"`
	SetupURL         string               `json:"setup_url,omitempty"`
	FreeTier         string               `json:"free_tier"`
	LimitModel       string               `json:"limit_model"`
	LimitConfidence  string               `json:"limit_confidence"`
	Status           ImplementationStatus `json:"implementation_status"`
	SetupGroup       string               `json:"setup_group,omitempty"`
	CredentialFields []CredentialField    `json:"credential_fields,omitempty"`
	Quota            QuotaTracking        `json:"quota"`
}

type CredentialField struct {
	Name        string `json:"name"`
	EnvVar      string `json:"env_var,omitempty"`
	Secret      bool   `json:"secret"`
	Required    bool   `json:"required"`
	Description string `json:"description,omitempty"`
}

type QuotaTracking struct {
	Mode          string   `json:"mode"`
	Headers       []string `json:"headers,omitempty"`
	Endpoint      string   `json:"endpoint,omitempty"`
	ManualLimit   string   `json:"manual_limit,omitempty"`
	Notes         string   `json:"notes,omitempty"`
	Source        string   `json:"source,omitempty"`
	CanPreflight  bool     `json:"can_preflight,omitempty"`
	CanObserve    bool     `json:"can_observe"`
	CanInferUsage bool     `json:"can_infer_usage"`
}

func Registry() []Provider {
	return withDerivedMetadata([]Provider{
		{ID: "jina", Name: "Jina Reader", Category: "extraction", Capabilities: []string{"extract.article", "fetch.url"}, AuthType: AuthAPIKey, OptionalAuth: true, EnvVar: "JINA_API_KEY", SetupURL: "https://jina.ai/reader/", FreeTier: "Reader access with API key; lower no-key limits", LimitModel: "RPM + token quota", LimitConfidence: "documented", Status: CapabilitySupported},
		{ID: "direct", Name: "Direct HTTP Fetch", Category: "local", Capabilities: []string{"fetch.url", "extract.article"}, AuthType: AuthNone, SetupURL: "", FreeTier: "Local HTTP client", LimitModel: "local/network only", LimitConfidence: "configured", Status: CapabilitySupported},
		{ID: "crossref", Name: "Crossref REST API", Category: "scholarly", Capabilities: []string{"search.scholar", "enrich.doi", "citations.doi"}, AuthType: AuthNone, SetupURL: "https://www.crossref.org/documentation/retrieve-metadata/rest-api/", FreeTier: "Free public REST API", LimitModel: "headers expose current limits; polite pool with mailto", LimitConfidence: "documented", Status: CapabilitySupported},
		{ID: "arxiv", Name: "arXiv API", Category: "scholarly", Capabilities: []string{"search.scholar"}, AuthType: AuthNone, SetupURL: "https://info.arxiv.org/help/api/index.html", FreeTier: "Free public API", LimitModel: "3 second delay between requests", LimitConfidence: "documented", Status: CapabilitySupported},
		{ID: "firecrawl", Name: "Firecrawl", Category: "extraction", Capabilities: []string{"fetch.url", "extract.article"}, AuthType: AuthAPIKey, EnvVar: "FIRECRAWL_API_KEY", SetupURL: "https://www.firecrawl.dev/", FreeTier: "1,000 credits/month", LimitModel: "monthly credits + concurrency", LimitConfidence: "documented", Status: CapabilitySupported},
		{ID: "scrapingant", Name: "ScrapingAnt", Category: "extraction", Capabilities: []string{"fetch.url", "extract.article"}, AuthType: AuthAPIKey, EnvVar: "SCRAPINGANT_API_KEY", SetupURL: "https://scrapingant.com/", FreeTier: "10,000 credits/month", LimitModel: "monthly credits per request option", LimitConfidence: "documented", Status: CapabilitySupported},
		{ID: "browserbase", Name: "Browserbase Fetch", Category: "extraction", Capabilities: []string{"fetch.url", "extract.article"}, AuthType: AuthAPIKey, EnvVar: "BROWSERBASE_API_KEY", SetupURL: "https://docs.browserbase.com/", FreeTier: "Free account includes fetch access under one API key", LimitModel: "free plan usage + endpoint limits", LimitConfidence: "documented", Status: CapabilitySupported},
		{ID: "openalex", Name: "OpenAlex", Category: "scholarly", Capabilities: []string{"search.scholar", "enrich.doi", "enrich.paper", "citations.doi"}, AuthType: AuthAPIKey, OptionalAuth: true, EnvVar: "OPENALEX_API_KEY", SetupURL: "https://openalex.org/", FreeTier: "Free public API; key improves quota", LimitModel: "daily allowance", LimitConfidence: "documented", Status: CapabilitySupported},
		{ID: "semantic_scholar", Name: "Semantic Scholar Academic Graph API", Category: "scholarly", Capabilities: []string{"search.scholar", "enrich.paper", "citations.doi"}, AuthType: AuthNone, SetupURL: "https://www.semanticscholar.org/product/api", FreeTier: "Public API with unauthenticated shared limits", LimitModel: "shared unauthenticated limits", LimitConfidence: "documented", Status: MetadataOnly},
		{ID: "pubmed", Name: "NCBI E-utilities / PubMed", Category: "scholarly", Capabilities: []string{"search.scholar"}, AuthType: AuthAPIKey, OptionalAuth: true, EnvVar: "NCBI_API_KEY", SetupURL: "https://www.ncbi.nlm.nih.gov/books/NBK25497/", FreeTier: "3 rps without key, 10 rps with key", LimitModel: "RPS", LimitConfidence: "documented", Status: CapabilitySupported},
		{ID: "datacite", Name: "DataCite REST API", Category: "scholarly", Capabilities: []string{"search.scholar"}, AuthType: AuthNone, SetupURL: "https://support.datacite.org/docs/api", FreeTier: "Free public REST API", LimitModel: "5-min windows", LimitConfidence: "documented", Status: CapabilitySupported},
		{ID: "opencitations", Name: "OpenCitations", Category: "citation graph", Capabilities: []string{"citations.doi"}, AuthType: AuthAPIKey, OptionalAuth: true, EnvVar: "OPENCITATIONS_ACCESS_TOKEN", SetupURL: "https://opencitations.net/", FreeTier: "Free public API; token encouraged", LimitModel: "token encouraged", LimitConfidence: "documented", Status: MetadataOnly},
		{ID: "unpaywall", Name: "Unpaywall", Category: "open access", Capabilities: []string{"enrich.doi"}, AuthType: AuthNone, SetupURL: "https://unpaywall.org/products/api", FreeTier: "Free public API; email required", LimitModel: "verify quota before high-volume", LimitConfidence: "documented", Status: MetadataOnly},
		{ID: "doaj", Name: "DOAJ API", Category: "scholarly", Capabilities: []string{"search.scholar"}, AuthType: AuthNone, SetupURL: "https://doaj.org/api/", FreeTier: "Free public API", LimitModel: "unknown", LimitConfidence: "unknown", Status: CapabilitySupported},
		{ID: "europepmc", Name: "Europe PMC REST API", Category: "scholarly", Capabilities: []string{"search.scholar"}, AuthType: AuthNone, SetupURL: "https://europepmc.org/RestfulWebService", FreeTier: "Free public API", LimitModel: "unknown", LimitConfidence: "unknown", Status: CapabilitySupported},
		{ID: "internet_archive", Name: "Internet Archive", Category: "archive", Capabilities: []string{"archive.lookup"}, AuthType: AuthNone, SetupURL: "https://archive.org/developers", FreeTier: "Free public APIs", LimitModel: "fair use", LimitConfidence: "documented", Status: CapabilitySupported},
		{ID: "commoncrawl", Name: "Common Crawl", Category: "archive", Capabilities: []string{"archive.lookup"}, AuthType: AuthNone, SetupURL: "https://commoncrawl.org/", FreeTier: "Free corpus", LimitModel: "public dataset/compute costs", LimitConfidence: "documented", Status: CapabilitySupported},
	})
}

func withDerivedMetadata(ps []Provider) []Provider {
	for i := range ps {
		p := &ps[i]
		if p.SetupGroup == "" {
			p.SetupGroup = setupGroup(*p)
		}
		if len(p.CredentialFields) == 0 {
			p.CredentialFields = credentialFields(*p)
		}
		if p.Quota.Mode == "" {
			p.Quota = quotaTracking(*p)
		}
	}
	return ps
}

func quotaTracking(p Provider) QuotaTracking {
	commonHeaders := []string{"Retry-After", "X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset", "RateLimit-Limit", "RateLimit-Remaining", "RateLimit-Reset"}
	switch p.ID {
	case "brave":
		return QuotaTracking{Mode: "headers", Headers: []string{"X-RateLimit-Limit", "X-RateLimit-Policy", "X-RateLimit-Remaining", "X-RateLimit-Reset"}, ManualLimit: "free credits monthly; plan-dependent QPS/monthly windows", CanObserve: true, CanInferUsage: true, Source: "https://api-dashboard.search.brave.com/documentation/guides/rate-limiting"}
	case "browserbase":
		return QuotaTracking{Mode: "headers", Headers: []string{"RateLimit-Limit", "RateLimit-Remaining", "RateLimit-Reset"}, ManualLimit: "plan-dependent search/fetch credits and request windows", CanObserve: true, CanInferUsage: true, Source: "https://docs.browserbase.com/guides/concurrency-rate-limits"}
	case "openalex":
		return QuotaTracking{Mode: "headers_and_endpoint", Headers: []string{"X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Credits-Used", "X-RateLimit-Reset"}, Endpoint: "https://api.openalex.org/rate-limit", ManualLimit: "free API key: 100,000 credits/day; max 100 rps", CanPreflight: true, CanObserve: true, CanInferUsage: true, Source: "https://docs.openalex.org/how-to-use-the-api/rate-limits-and-authentication"}
	case "firecrawl":
		return QuotaTracking{Mode: "manual_and_429", Headers: commonHeaders, ManualLimit: "monthly credits plus concurrency; 429 on rate limit", CanObserve: true, CanInferUsage: true, Source: "https://docs.firecrawl.dev/api-reference/introduction"}
	case "semantic_scholar":
		return QuotaTracking{Mode: "manual_and_429", ManualLimit: "public endpoints share unauthenticated limit; API-key introductory limit is 1 rps", CanObserve: false, CanInferUsage: true, Source: "https://www.semanticscholar.org/product/api"}
	case "pubmed":
		return QuotaTracking{Mode: "headers", Headers: []string{"X-RateLimit-Limit", "X-RateLimit-Remaining"}, ManualLimit: "3 rps without key, 10 rps with key", CanObserve: true, CanInferUsage: true, Source: "https://www.ncbi.nlm.nih.gov/books/NBK25497/"}
	case "crossref":
		return QuotaTracking{Mode: "headers", Headers: commonHeaders, ManualLimit: "polite pool and fair-use limits; headers may expose current windows", CanObserve: true, CanInferUsage: true, Source: "https://www.crossref.org/documentation/retrieve-metadata/rest-api/"}
	case "arxiv":
		return QuotaTracking{Mode: "manual", ManualLimit: "wait at least 3 seconds between API requests", CanObserve: false, CanInferUsage: true, Source: "https://info.arxiv.org/help/api/index.html"}
	case "datacite", "doaj", "europepmc", "internet_archive", "commoncrawl", "direct":
		return QuotaTracking{Mode: "fair_use", ManualLimit: p.LimitModel, CanObserve: false, CanInferUsage: true, Source: p.SetupURL}
	default:
		return QuotaTracking{Mode: "manual", Headers: commonHeaders, ManualLimit: p.LimitModel, CanObserve: false, CanInferUsage: true, Source: p.SetupURL}
	}
}

func setupGroup(p Provider) string {
	if p.Status == LegacyOptional {
		return "legacy_optional"
	}
	if p.AuthType == AuthNone {
		return "no_key"
	}
	if p.AuthType == AuthOAuth {
		return "oauth"
	}
	if p.ID == "scrapingant" || p.ID == "browserbase" || p.ID == "firecrawl" || p.ID == "jina" {
		return "extraction"
	}
	return "easy_key"
}

func credentialFields(p Provider) []CredentialField {
	switch p.ID {
	case "crossref":
		return []CredentialField{{Name: "contact_email", EnvVar: "FORAGE_CONTACT_EMAIL", Secret: false, Required: false, Description: "Contact email for Crossref polite pool user-agent"}}
	case "unpaywall":
		return []CredentialField{{Name: "email", EnvVar: "UNPAYWALL_EMAIL", Secret: false, Required: true, Description: "Contact email required by Unpaywall API requests"}}
	}
	if p.AuthType == AuthAPIKey {
		return []CredentialField{{Name: "api_key", EnvVar: p.EnvVar, Secret: true, Required: !p.OptionalAuth, Description: p.Name + " API key"}}
	}
	return nil
}

func ByID(id string) (Provider, bool) {
	for _, p := range Registry() {
		if p.ID == id {
			return p, true
		}
	}
	return Provider{}, false
}

func Sorted() []Provider {
	ps := Registry()
	sort.Slice(ps, func(i, j int) bool { return ps[i].ID < ps[j].ID })
	return ps
}
