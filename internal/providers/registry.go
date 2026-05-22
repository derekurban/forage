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
}

type CredentialField struct {
	Name        string `json:"name"`
	EnvVar      string `json:"env_var,omitempty"`
	Secret      bool   `json:"secret"`
	Required    bool   `json:"required"`
	Description string `json:"description,omitempty"`
}

func Registry() []Provider {
	return withDerivedMetadata([]Provider{
		{ID: "brave", Name: "Brave Search API", Category: "web search", Capabilities: []string{"search.web", "search.news", "search.images", "answers.llm"}, AuthType: AuthAPIKey, EnvVar: "BRAVE_SEARCH_API_KEY", SetupURL: "https://api-dashboard.search.brave.com/", FreeTier: "$5 free credits monthly", LimitModel: "monthly credits + endpoint request cost", LimitConfidence: "documented", Status: CapabilitySupported},
		{ID: "exa", Name: "Exa", Category: "AI-native search", Capabilities: []string{"search.web"}, AuthType: AuthAPIKey, EnvVar: "EXA_API_KEY", SetupURL: "https://dashboard.exa.ai/", FreeTier: "1,000 requests/month free", LimitModel: "monthly requests", LimitConfidence: "documented", Status: CapabilitySupported},
		{ID: "tavily", Name: "Tavily", Category: "AI search", Capabilities: []string{"search.web"}, AuthType: AuthAPIKey, EnvVar: "TAVILY_API_KEY", SetupURL: "https://app.tavily.com/", FreeTier: "1,000 API credits/month free", LimitModel: "monthly credits", LimitConfidence: "documented", Status: CapabilitySupported},
		{ID: "jina", Name: "Jina Reader/Search", Category: "search/extraction", Capabilities: []string{"search.web", "extract.article", "fetch.url"}, AuthType: AuthAPIKey, OptionalAuth: true, EnvVar: "JINA_API_KEY", SetupURL: "https://jina.ai/reader/", FreeTier: "Reader 500 RPM with key; search 100 RPM; lower no-key limits", LimitModel: "RPM + token quota", LimitConfidence: "documented", Status: CapabilitySupported},
		{ID: "direct", Name: "Direct HTTP Fetch", Category: "local", Capabilities: []string{"fetch.url", "extract.article"}, AuthType: AuthNone, SetupURL: "", FreeTier: "Local HTTP client", LimitModel: "local/network only", LimitConfidence: "configured", Status: CapabilitySupported},
		{ID: "hackernews", Name: "Hacker News API", Category: "platform", Capabilities: []string{"search.platform"}, AuthType: AuthNone, SetupURL: "https://github.com/HackerNews/API", FreeTier: "Free public API", LimitModel: "no documented current rate limit", LimitConfidence: "documented", Status: CapabilitySupported},
		{ID: "crossref", Name: "Crossref REST API", Category: "scholarly", Capabilities: []string{"search.scholar", "enrich.citations"}, AuthType: AuthNone, SetupURL: "https://www.crossref.org/documentation/retrieve-metadata/rest-api/", FreeTier: "Free public REST API", LimitModel: "headers expose current limits; polite pool with mailto", LimitConfidence: "documented", Status: CapabilitySupported},
		{ID: "arxiv", Name: "arXiv API", Category: "scholarly", Capabilities: []string{"search.scholar"}, AuthType: AuthNone, SetupURL: "https://info.arxiv.org/help/api/index.html", FreeTier: "Free public API", LimitModel: "3 second delay between requests", LimitConfidence: "documented", Status: CapabilitySupported},
		{ID: "serpapi", Name: "SerpApi", Category: "SERP", Capabilities: []string{"search.web", "search.news", "search.scholar"}, AuthType: AuthAPIKey, EnvVar: "SERPAPI_API_KEY", SetupURL: "https://serpapi.com/", FreeTier: "250 searches/month", LimitModel: "monthly searches + hourly throughput", LimitConfidence: "documented", Status: MetadataOnly},
		{ID: "serpstack", Name: "serpstack", Category: "SERP", Capabilities: []string{"search.web"}, AuthType: AuthAPIKey, EnvVar: "SERPSTACK_API_KEY", SetupURL: "https://serpstack.com/", FreeTier: "100 searches/month", LimitModel: "monthly searches", LimitConfidence: "documented", Status: MetadataOnly},
		{ID: "firecrawl", Name: "Firecrawl", Category: "extraction/crawl", Capabilities: []string{"fetch.url", "extract.article", "crawl.site"}, AuthType: AuthAPIKey, EnvVar: "FIRECRAWL_API_KEY", SetupURL: "https://www.firecrawl.dev/", FreeTier: "1,000 credits/month", LimitModel: "monthly credits + concurrency", LimitConfidence: "documented", Status: CapabilitySupported},
		{ID: "scrapingant", Name: "ScrapingAnt", Category: "render/scrape", Capabilities: []string{"fetch.url", "extract.article", "render.browser"}, AuthType: AuthAPIKey, EnvVar: "SCRAPINGANT_API_KEY", SetupURL: "https://scrapingant.com/", FreeTier: "10,000 credits/month", LimitModel: "monthly credits per request option", LimitConfidence: "documented", Status: CapabilitySupported},
		{ID: "apify", Name: "Apify", Category: "scraping/crawl", Capabilities: []string{"extract.article", "crawl.site", "render.browser"}, AuthType: AuthAPIKey, EnvVar: "APIFY_API_KEY", SetupURL: "https://apify.com/", FreeTier: "$5/month platform usage", LimitModel: "compute units", LimitConfidence: "documented", Status: MetadataOnly},
		{ID: "browserless", Name: "Browserless", Category: "browser", Capabilities: []string{"render.browser"}, AuthType: AuthAPIKey, EnvVar: "BROWSERLESS_API_KEY", SetupURL: "https://browserless.io/", FreeTier: "1,000 units/month", LimitModel: "browser time units + concurrency", LimitConfidence: "documented", Status: MetadataOnly},
		{ID: "browserbase", Name: "Browserbase Search/Fetch", Category: "web data", Capabilities: []string{"search.web", "fetch.url", "extract.article"}, AuthType: AuthAPIKey, EnvVar: "BROWSERBASE_API_KEY", SetupURL: "https://docs.browserbase.com/", FreeTier: "Free account includes search/fetch access under one API key", LimitModel: "free plan usage + endpoint limits", LimitConfidence: "documented", Status: CapabilitySupported},
		{ID: "openalex", Name: "OpenAlex", Category: "scholarly", Capabilities: []string{"search.scholar", "resolve.identity", "enrich.citations"}, AuthType: AuthAPIKey, EnvVar: "OPENALEX_API_KEY", SetupURL: "https://openalex.org/", FreeTier: "Free key/account for API use", LimitModel: "daily allowance", LimitConfidence: "documented", Status: CapabilitySupported},
		{ID: "semantic_scholar", Name: "Semantic Scholar Academic Graph API", Category: "scholarly", Capabilities: []string{"search.scholar", "enrich.citations"}, AuthType: AuthNone, SetupURL: "https://www.semanticscholar.org/product/api", FreeTier: "Public API with unauthenticated shared limits", LimitModel: "shared unauthenticated limits", LimitConfidence: "documented", Status: MetadataOnly},
		{ID: "pubmed", Name: "NCBI E-utilities / PubMed", Category: "scholarly", Capabilities: []string{"search.scholar"}, AuthType: AuthAPIKey, OptionalAuth: true, EnvVar: "NCBI_API_KEY", SetupURL: "https://www.ncbi.nlm.nih.gov/books/NBK25497/", FreeTier: "3 rps without key, 10 rps with key", LimitModel: "RPS", LimitConfidence: "documented", Status: CapabilitySupported},
		{ID: "datacite", Name: "DataCite REST API", Category: "scholarly", Capabilities: []string{"search.scholar"}, AuthType: AuthNone, SetupURL: "https://support.datacite.org/docs/api", FreeTier: "Free public REST API", LimitModel: "5-min windows", LimitConfidence: "documented", Status: CapabilitySupported},
		{ID: "wikidata", Name: "Wikidata SPARQL", Category: "entity graph", Capabilities: []string{"resolve.identity"}, AuthType: AuthNone, SetupURL: "https://www.wikidata.org/wiki/Wikidata:SPARQL_query_service", FreeTier: "Free public endpoint", LimitModel: "fair use/query limits", LimitConfidence: "documented", Status: MetadataOnly},
		{ID: "opencitations", Name: "OpenCitations", Category: "citation graph", Capabilities: []string{"enrich.citations"}, AuthType: AuthAPIKey, OptionalAuth: true, EnvVar: "OPENCITATIONS_ACCESS_TOKEN", SetupURL: "https://opencitations.net/", FreeTier: "Free public API; token encouraged", LimitModel: "token encouraged", LimitConfidence: "documented", Status: MetadataOnly},
		{ID: "orcid", Name: "ORCID Public API", Category: "identity", Capabilities: []string{"resolve.identity"}, AuthType: AuthOAuth, EnvVar: "ORCID_CLIENT_ID", SetupURL: "https://info.orcid.org/documentation/api-tutorials/", FreeTier: "Public API credentials", LimitModel: "public API terms", LimitConfidence: "documented", Status: MetadataOnly},
		{ID: "unpaywall", Name: "Unpaywall", Category: "open access", Capabilities: []string{"enrich.citations"}, AuthType: AuthNone, SetupURL: "https://unpaywall.org/products/api", FreeTier: "Free public API; email required", LimitModel: "verify quota before high-volume", LimitConfidence: "documented", Status: MetadataOnly},
		{ID: "doaj", Name: "DOAJ API", Category: "scholarly", Capabilities: []string{"search.scholar"}, AuthType: AuthNone, SetupURL: "https://doaj.org/api/", FreeTier: "Free public API", LimitModel: "unknown", LimitConfidence: "unknown", Status: CapabilitySupported},
		{ID: "europepmc", Name: "Europe PMC REST API", Category: "scholarly", Capabilities: []string{"search.scholar"}, AuthType: AuthNone, SetupURL: "https://europepmc.org/RestfulWebService", FreeTier: "Free public API", LimitModel: "unknown", LimitConfidence: "unknown", Status: CapabilitySupported},
		{ID: "gdelt", Name: "GDELT", Category: "news corpus", Capabilities: []string{"search.news", "archive.lookup"}, AuthType: AuthNone, SetupURL: "https://www.gdeltproject.org/", FreeTier: "Free/open access", LimitModel: "fair use/open corpus", LimitConfidence: "documented", Status: MetadataOnly},
		{ID: "internet_archive", Name: "Internet Archive", Category: "archive", Capabilities: []string{"archive.lookup"}, AuthType: AuthNone, SetupURL: "https://archive.org/developers", FreeTier: "Free public APIs", LimitModel: "fair use", LimitConfidence: "documented", Status: CapabilitySupported},
		{ID: "commoncrawl", Name: "Common Crawl", Category: "corpus", Capabilities: []string{"archive.lookup"}, AuthType: AuthNone, SetupURL: "https://commoncrawl.org/", FreeTier: "Free corpus", LimitModel: "public dataset/compute costs", LimitConfidence: "documented", Status: CapabilitySupported},
		{ID: "guardian", Name: "The Guardian Open Platform", Category: "news", Capabilities: []string{"search.news"}, AuthType: AuthAPIKey, EnvVar: "GUARDIAN_API_KEY", SetupURL: "https://open-platform.theguardian.com/", FreeTier: "500 calls/day non-commercial", LimitModel: "daily + 1 call/s", LimitConfidence: "documented", Status: CapabilitySupported},
		{ID: "currents", Name: "Currents API", Category: "news", Capabilities: []string{"search.news"}, AuthType: AuthAPIKey, EnvVar: "CURRENTS_API_KEY", SetupURL: "https://currentsapi.services/", FreeTier: "1,000 requests/day", LimitModel: "daily requests", LimitConfidence: "documented", Status: MetadataOnly},
		{ID: "newsapi", Name: "NewsAPI.org", Category: "news", Capabilities: []string{"search.news"}, AuthType: AuthAPIKey, EnvVar: "NEWSAPI_API_KEY", SetupURL: "https://newsapi.org/", FreeTier: "100 requests/day", LimitModel: "daily requests + delayed data", LimitConfidence: "documented", Status: CapabilitySupported},
		{ID: "gnews", Name: "GNews API", Category: "news", Capabilities: []string{"search.news"}, AuthType: AuthAPIKey, EnvVar: "GNEWS_API_KEY", SetupURL: "https://gnews.io/", FreeTier: "100 requests/day", LimitModel: "daily requests + delay", LimitConfidence: "documented", Status: CapabilitySupported},
		{ID: "mediastack", Name: "Mediastack", Category: "news", Capabilities: []string{"search.news"}, AuthType: AuthAPIKey, EnvVar: "MEDIASTACK_API_KEY", SetupURL: "https://mediastack.com/", FreeTier: "100 calls/month", LimitModel: "monthly calls", LimitConfidence: "documented", Status: MetadataOnly},
		{ID: "worldnews", Name: "World News API", Category: "news", Capabilities: []string{"search.news"}, AuthType: AuthAPIKey, EnvVar: "WORLDNEWS_API_KEY", SetupURL: "https://worldnewsapi.com/", FreeTier: "50 points/day", LimitModel: "daily points", LimitConfidence: "documented", Status: MetadataOnly},
		{ID: "forem", Name: "Forem / DEV API", Category: "platform", Capabilities: []string{"search.platform"}, AuthType: AuthNone, SetupURL: "https://developers.forem.com/api/v1", FreeTier: "Free public API", LimitModel: "unknown", LimitConfidence: "unknown", Status: CapabilitySupported},
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
	}
	return ps
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
	if p.ID == "browserless" || p.ID == "scrapingant" || p.ID == "apify" || p.ID == "browserbase" {
		return "browser_render"
	}
	return "easy_key"
}

func credentialFields(p Provider) []CredentialField {
	switch p.ID {
	case "brave":
		return []CredentialField{
			{Name: "search_api_key", EnvVar: "BRAVE_SEARCH_API_KEY", Secret: true, Required: true, Description: "Brave Search API key for web/news/image search"},
			{Name: "answers_api_key", EnvVar: "BRAVE_ANSWERS_API_KEY", Secret: true, Required: false, Description: "Brave Answers API key for future LLM answer endpoints"},
		}
	case "orcid":
		return []CredentialField{
			{Name: "client_id", EnvVar: "ORCID_CLIENT_ID", Secret: false, Required: true, Description: "ORCID public API client ID"},
			{Name: "client_secret", EnvVar: "ORCID_CLIENT_SECRET", Secret: true, Required: true, Description: "ORCID public API client secret"},
		}
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
