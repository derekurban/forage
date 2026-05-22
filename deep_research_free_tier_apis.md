# Deep Research Agent: APIs & Platforms With Continual Free Usage

**Last verified:** 2026-05-21  
**Scope:** Web search, SERP APIs, AI-native search, page fetching, article/blog/news discovery, scraping/crawling/extraction, browser APIs, scholarly/research metadata, citation graphs, open web/news corpora, and public content APIs.

## Inclusion rule

This file includes services that appear to support **continual free usage over time**: recurring monthly/daily quotas, free-forever tiers, free public APIs, or public corpora with ongoing access.

It excludes providers whose “free” usage is only a **one-time signup grant**, **temporary trial**, or **expiring credit**, unless they also have a separate recurring free tier.

## Important architecture note

For a deep research agent, do **not** treat “search,” “fetch,” “extract,” and “corpus access” as one problem.

A robust research stack usually needs:

1. **Discovery** — search APIs, SERPs, news indexes, scholarly indexes.
2. **Fetch/extraction** — URL-to-markdown/text, JS rendering, article extraction, PDF handling.
3. **Corpus APIs** — OpenAlex, PubMed, Crossref, Semantic Scholar, GDELT, Common Crawl, Internet Archive, etc.
4. **Source-specific APIs** — WordPress, Blogger, Forem/DEV, Hacker News, Guardian, Reddit, etc.
5. **Compliance/cache layer** — canonical URL, date seen, source license/terms, extraction method, robots/ToS flags, deduplication, provenance.

---

# 1. General web search, SERP, and AI-native search APIs

| Service | Website / pricing source | Category | Continual free usage | What it gives you | Restrictions / notes | Paid pricing details found |
|---|---|---|---|---|---|---|
| **Exa** | [exa.ai/pricing](https://exa.ai/pricing) | AI-native web search, contents, deep search, agent search | **1,000 requests/month free** | Search tool calls for agents, webpage text/highlights, low-latency search, contents endpoint, deep search, async agents | Good fit for AI-agent retrieval rather than raw SERP cloning | Search: **$7/1k requests**; Deep Search: **$12–15/1k**; Contents: **$1/1k pages per content type**; Agent: **$0.025–$2/run** |
| **Tavily** | [tavily.com/pricing](https://www.tavily.com/pricing) | AI search API for agents | **1,000 API credits/month free**; no credit card required | Search/research-oriented API for AI agents | Credit usage depends on endpoint/features | PAYG listed at **$0.008/credit** |
| **SerpApi** | [serpapi.com/pricing](https://serpapi.com/pricing) | SERP API / search result extraction | **250 searches/month free**; **50 throughput/hour** | Google, Google News, Google Scholar, Google Patents, Bing, DuckDuckGo, YouTube, Amazon, Maps, Shopping, etc. | Successful searches count; cached/failed/errored searches do not count | Starter: **$25/month**, **1,000 searches/month**, **200 throughput/hour** |
| **serpstack** | [serpstack.com/pricing](https://serpstack.com/pricing) | Google SERP API | **100 searches/month free forever** | Google search result JSON; supports many Google result types | Free plan is basic API functionality; JSON only; no support | Basic: **$29.99/month**, **5,000 searches/month** |
| **Brave Search API** | [Brave Search API pricing](https://api-dashboard.search.brave.com/documentation/pricing) | Search API / news / images / videos / LLM context | **$5 free credits every month**, automatically applied. At **$5/1k search requests**, this is roughly **1,000 search requests/month** for search endpoints | Web search, news search, image/video search, LLM context, autosuggest/spellcheck, answer APIs | Credit-based. Answers have separate query + token pricing. | Search: **$5/1k requests**; Answers: **$4/1k queries + $5/1M input tokens + $5/1M output tokens**; Autosuggest/spellcheck: **$5/10k requests** |
| **Jina Reader Search (`s.jina.ai`)** | [jina.ai/reader](https://jina.ai/reader/) | AI-friendly search endpoint | Free rate-limited usage: **100 RPM** for search endpoint, with or without free API key | Search endpoint designed to return LLM-friendly web results | Search and reader token usage share quota model; not a classic monthly SERP allowance | Premium raises search to **1,000 RPM** |
| **Google Custom Search JSON API / Programmable Search** | [Google Custom Search JSON API docs](https://developers.google.com/custom-search/v1/overview) | Programmable web/image search API | **100 queries/day free** | JSON web/image results from configured search engines | **Legacy caveat:** Google states Custom Search JSON API is closed to new customers; existing customers can use it until **2027-01-01** | Paid usage: **$5/1,000 queries**, up to 10,000/day |

---

# 2. Page fetching, extraction, scraping, crawling, and browser APIs

| Service | Website / pricing source | Category | Continual free usage | What it gives you | Restrictions / notes | Paid pricing details found |
|---|---|---|---|---|---|---|
| **Jina Reader (`r.jina.ai`)** | [jina.ai/reader](https://jina.ai/reader/) | URL-to-markdown/text extraction | Free rate-limited usage: **20 RPM without key**, **500 RPM with free API key** | Converts URLs to LLM-friendly markdown; supports web pages, PDFs, images, selectors, cache controls, and search endpoint | Great first-pass extraction layer for blogs/articles/docs. Free but rate-limited; use responsibly. | Premium Reader rate limit listed as **5,000 RPM** |
| **Firecrawl** | [firecrawl.dev/pricing](https://www.firecrawl.dev/pricing) | Scrape, crawl, map, search, extract | **1,000 credits/month free**; no credit card required | Scraping, crawling, URL mapping, search, extraction, LLM-ready markdown | Free plan: about **1,000 pages/month**, **2 concurrent requests**, low rate limits. Endpoint credits vary: Scrape/Crawl/Map ≈ 1/page; Search ≈ 2/10 results | Paid plans scale credits/concurrency; endpoint credit table published |
| **ScrapingAnt** | [scrapingant.com](https://scrapingant.com/) | Scraping API, browser rendering, proxies, extraction | **10,000 free credits every month**; no credit card required | JS-rendered scraping, rotating proxies, markdown/text extraction, MCP tooling | Default Chrome-rendered request with standard proxies costs **10 credits**; requests can cost **1–25 credits** depending options | Paid plans are credit-based |
| **Apify** | [apify.com/pricing](https://apify.com/pricing) | Scraping/crawling platform, actor marketplace, proxies | **$5/month free platform usage**; no credit card required | Actor marketplace, custom crawlers, proxy options, scheduled jobs, datasets, MCP/agent tooling | Free plan includes limited platform spend and concurrency; overages blocked until next billing cycle | Compute unit listed at **$0.20/CU**; free includes **$5 prepaid usage/month** |
| **Diffbot** | [diffbot.com/pricing](https://www.diffbot.com/pricing/) | Article extraction, crawl, natural language, knowledge graph | **10,000 credits/month free forever**; **5 calls/min** | Automatic article/product/page extraction, Crawl, Bulk Extract, Natural Language, Knowledge Graph Search/Enhance | One page extraction is typically **1 credit**; exporting one Knowledge Graph entity record is **25 credits** | Free monthly allotment resets each billing period |
| **Browserless** | [browserless.io/pricing](https://www.browserless.io/pricing/) | Hosted browser automation / Chrome API | **1,000 units/month free**; no credit card required | Remote browsers for Puppeteer/Playwright-style automation, screenshots, JS-rendered sites, captchas/proxies/features | Free plan: **2 concurrent browsers**, **1-minute max session**, logs/sessions kept 1 day. One unit = 30 seconds browser time. | Proxy/captcha features consume additional units; paid plans scale units/concurrency |

---

# 3. News, articles, blogs, and media APIs

| Service | Website / pricing source | Category | Continual free usage | What it gives you | Restrictions / notes | Paid pricing details found |
|---|---|---|---|---|---|---|
| **Currents API** | [currentsapi.services/pricing](https://currentsapi.services/en/pricing) | News API | **1,000 requests/day free** | Headlines, partial article access, news search, 3 months history | Free tier gives partial article access, not full text | Paid plans increase request volume/history/fullness |
| **NewsAPI.org** | [newsapi.org/pricing](https://newsapi.org/pricing) | News search/headlines API | **100 requests/day free** | Article search and top headlines | Developer/testing only; **24-hour delay**; search articles up to **1 month old**; no full article content | Paid plans for production/commercial use |
| **GNews API** | [gnews.io/pricing](https://gnews.io/pricing) | News API | **100 requests/day free** | Up to **10 articles/request**, 30-day history, all sources | Free plan is for development/testing/non-commercial; **12-hour delay**; daily reset at 00:00 UTC | Paid plans unlock more articles/request, freshness, commercial usage |
| **Mediastack** | [mediastack.com/product](https://mediastack.com/product) | News/blogs API | **100 calls/month free forever** | News data from global sources and blogs, 50+ countries, 13 languages | Very small quota; delayed data; no support | Paid plans unlock larger monthly quotas and real-time data |
| **World News API** | [worldnewsapi.com/pricing](https://worldnewsapi.com/pricing) | News API | **50 points/day free** | News search, article metadata, 1-month history | Free plan requires backlink; **1 request/s**, **1 concurrent request** | Paid plans increase points/history/concurrency |
| **The Guardian Open Platform** | [open-platform.theguardian.com/access](https://open-platform.theguardian.com/access/) | Publisher API | **500 calls/day**, **1 call/s**, non-commercial developer key | Guardian article metadata and article text | Free key is non-commercial. Commercial use, AI/model training, and text mining require a commercial key/custom terms. | Commercial terms are custom |
| **GDELT Project** | [gdeltproject.org](https://www.gdeltproject.org/) | Open global news/media/event corpus | Free/open access; updates every **15 minutes** | Global print, broadcast, and web news monitoring in 100+ languages; historical archive back to 1979 for event datasets | Better as a media-intelligence and news-discovery corpus than a clean full-text news API | Free/open data; BigQuery access also available |
| **WordPress REST API** | [developer.wordpress.org/rest-api](https://developer.wordpress.org/rest-api/) | Blog/CMS source API | Free public access where enabled by each WordPress site | Posts, pages, taxonomies, media, search, comments, etc. | Public content is generally publicly accessible through the REST API; private/protected content requires auth. Site owners/plugins/hosts may limit or disable access. | No platform pricing; per-site infrastructure/rate limits apply |
| **WordPress.com REST API** | [developer.wordpress.com/docs/api](https://developer.wordpress.com/docs/api/) | WordPress.com / Jetpack source API | Free access to public endpoints; authenticated endpoints require OAuth | WordPress.com and Jetpack-connected site data: posts, comments, taxonomy, media, users, stats, etc. | Subject to Automattic API terms and responsible-use/rate limiting guidelines | No simple fixed free quota surfaced |
| **Blogger API v3** | [developers.google.com/blogger/docs/3.0/using](https://developers.google.com/blogger/docs/3.0/using) | Blog platform API | Free public blog retrieval with API key | Public Blogger/Blogspot blogs, posts, pages, comments, search | Public blog requests require API key but not user auth; private blogs require auth. Google Cloud quotas may apply. | Google API quotas apply; no simple paid public pricing surfaced |
| **Forem / DEV API** | [developers.forem.com/api/v1](https://developers.forem.com/api/v1) | Developer/blog/community articles | Free public API access | DEV/Forem published articles, latest articles, user/org articles, tags, comments | Exact public quota was not clearly surfaced; API-key auth required for user-specific endpoints | No simple paid public pricing surfaced |
| **Hacker News API** | [github.com/HackerNews/API](https://github.com/HackerNews/API) | Tech article/discussion discovery API | Free public API; official docs state **currently no rate limit** | Near-real-time HN stories, comments, jobs, Ask HN, Show HN, poll items, external URLs | API is intentionally simple/awkward; client should handle trees and extra fields gracefully | Free public API |
| **Reddit Data API** | [Reddit Data API Wiki](https://support.reddithelp.com/hc/en-us/articles/16160319875092-Reddit-Data-API-Wiki) | Discussion/social/news aggregation API | Free access usage for eligible apps: **100 QPM per OAuth client ID** | Reddit posts/comments/subreddit data and discussion around articles/blogs | OAuth required; user-agent required; deleted content/data retention obligations; commercial/research eligibility and terms matter | Commercial/large-scale access may require separate terms |

---

# 4. Scholarly research, papers, citations, white papers, and open corpora

| Service | Website / docs source | Category | Continual free usage | What it gives you | Restrictions / notes | Paid pricing details found |
|---|---|---|---|---|---|---|
| **OpenAlex** | [openalex.org/pricing](https://openalex.org/pricing) | Scholarly metadata / papers / authors / institutions / concepts | Free API usage with **$1/day free allowance**; examples listed include **1,000 search requests/day**, **10,000 list+filter requests/day**, **100 PDF/content downloads/day**, and unlimited single-entity requests within free allowance | Massive scholarly graph: works, authors, sources, institutions, concepts, funders, publishers, topics | Excellent first scholarly discovery layer | Higher paid tiers available for larger scale |
| **Semantic Scholar Academic Graph API** | [semanticscholar.org/product/api](https://www.semanticscholar.org/product/api) | Scholarly search/metadata/citations/recommendations/datasets | Most endpoints public without auth; unauthenticated traffic is rate-limited to **1,000 requests/s shared across all unauthenticated users**; API keys start at **1 RPS** | Papers, authors, citations, venues, embeddings, recommendations, downloadable datasets | Shared unauthenticated limit can throttle during heavy use; API key recommended | Higher rate limits may be available by request |
| **NCBI E-utilities / PubMed** | [NCBI E-utilities docs](https://www.ncbi.nlm.nih.gov/books/NBK25497/) | Biomedical literature/search API | **3 requests/s without key**, **10 requests/s with API key** | Programmatic access to Entrez databases including PubMed and biomedical literature metadata | Large jobs should run off-peak; higher rates can be requested | Free public API |
| **Crossref REST API** | [Crossref REST API tips](https://www.crossref.org/documentation/retrieve-metadata/rest-api/tips-for-using-the-crossref-rest-api/) | DOI metadata / scholarly metadata | Free public REST API; use polite pool with `mailto`; rate-limit headers show current limits | DOI metadata, works, funders, members, journals, references where available | Use HTTPS and contact email for polite pool; production/SLA use can require Crossref Plus | Crossref Plus exists for production/SLA needs |
| **arXiv API** | [arxiv.org/help/api](https://info.arxiv.org/help/api/index.html) | Preprint search API | Free public API; requires **3-second delay** between requests | Search and retrieve arXiv e-print metadata | Max result-window guidance: use slices; OAI-PMH for bulk harvesting | Free public API |
| **DataCite REST API** | [support.datacite.org/docs/api](https://support.datacite.org/docs/api) | DOI metadata / research objects | Free public REST API; rate examples: **500 requests/5 min/IP** unidentified, **1,000/5 min/IP** identified, **3,000/5 min/IP** authenticated | DOI metadata for datasets, research outputs, organizations, repositories | Authentication/identification improves quota | Free public API; member services separate |
| **Wikidata SPARQL Query Service** | [Wikidata SPARQL query service](https://www.wikidata.org/wiki/Wikidata:SPARQL_query_service) | Structured knowledge graph / entity graph / scholarly subset | Free public SPARQL endpoint | Entity relationships, identifiers, organizations, people, publications, scholarly subset, CSV/JSON/XML/TSV output | Query limits and fair-use behavior apply; complex queries may timeout | Free public endpoint |
| **OpenCitations Index APIs** | [api.opencitations.net/index/v1](https://api.opencitations.net/index/v1) | Open citation graph | Free public REST APIs; access token encouraged for applications | Citation counts, references, citations by DOI, Open Citation Identifiers | Legacy v1 exists; latest v2 recommended for new work | Free/open citation infrastructure |
| **ORCID Public API** | [ORCID public API tutorial](https://info.orcid.org/documentation/api-tutorials/api-tutorial-read-data-on-a-record/) | Researcher identity / author records | Anyone can register for read-only Public API credentials; public read tokens are long-lived | Public ORCID records, researcher profiles, works, affiliations, identifiers | Public API terms limit certain uses; member API needed for higher-volume/commercial/member workflows | Public API free; Member API requires membership |
| **Internet Archive Developer APIs** | [archive.org/developers](https://archive.org/developers/) | Historical web, books, media, metadata, Wayback-like data | Free public access to Internet Archive APIs and data | Archive metadata, items, files, WARC/web archive concepts, OCR/PDF-related resources, historical captures | Respect automated access rules and rights issues; not all items are equally reusable | Free public infrastructure |
| **Common Crawl** | [commoncrawl.org/overview](https://commoncrawl.org/overview) | Open web crawl corpus | Free access to petabyte-scale web crawl data on AWS public datasets | Raw WARC data, metadata, text extracts, URL index; crawls since 2008 | Not real-time; requires data engineering. AWS access is free for public dataset access, but compute/egress choices can cost money. | Free corpus; compute/storage costs are yours |
| **GDELT** | [gdeltproject.org](https://www.gdeltproject.org/) | News/event/media corpus | Free/open access; updated every **15 minutes** | Global news/event database, themes, entities, geographic/event data, multilingual monitoring | Strong for media intelligence and historical news analysis | Free/open data |
| **Unpaywall API** | [unpaywall.org/products/api](https://unpaywall.org/products/api) | Open-access scholarly full-text locator | Continual free public API access; email required in requests | Finds legal open-access versions of scholarly papers by DOI | The fetched page was JS-only during verification, so exact current quota should be rechecked before high-volume use | Historically free/public; verify current daily quota before production |
| **DOAJ API** | [doaj.org/api](https://doaj.org/api/) | Open-access journal/article metadata | Free public API access | Directory of Open Access Journals metadata and article records | Main docs fetch returned 403 during verification; exact current quota should be rechecked | Free/open metadata API |
| **Europe PMC REST API** | [europepmc.org/RestfulWebService](https://europepmc.org/RestfulWebService) | Biomedical/life sciences literature | Free public REST API access | Literature search, abstracts, full-text links, grants, annotations for Europe PMC corpus | Direct fetch failed during verification; exact current rate policy should be rechecked | Free public API |

---

# 5. Public/open source or self-hosted complements

These are not hosted free-tier SaaS APIs, but they are often part of a serious research-agent stack because they can lower external API cost.

| Tool / project | Website | Category | Free usage model | Use in architecture | Notes |
|---|---|---|---|---|---|
| **Scrapy** | [scrapy.org](https://scrapy.org/) | Open-source crawling framework | Free/self-hosted | Build custom crawlers, spiders, pipelines | You pay infrastructure/proxy/browser costs |
| **Crawlee** | [crawlee.dev](https://crawlee.dev/) | Open-source crawler framework | Free/self-hosted | Modern JS/TS crawling, Playwright/Puppeteer integration | Apify ecosystem; self-host possible |
| **Playwright** | [playwright.dev](https://playwright.dev/) | Browser automation | Free/self-hosted | JS rendering, screenshots, controlled browser sessions | You pay browser compute/proxies |
| **Puppeteer** | [pptr.dev](https://pptr.dev/) | Browser automation | Free/self-hosted | Chrome automation and page rendering | You pay infrastructure |
| **Readability.js** | [github.com/mozilla/readability](https://github.com/mozilla/readability) | Article extraction library | Free/self-hosted | Extract main article body from HTML | Good local fallback after fetching pages |
| **trafilatura** | [trafilatura.readthedocs.io](https://trafilatura.readthedocs.io/) | Web text extraction library | Free/self-hosted | HTML-to-clean-text extraction | Useful for local batch extraction |
| **newspaper3k / newspaper4k** | [newspaper.readthedocs.io](https://newspaper.readthedocs.io/) | Article extraction library | Free/self-hosted | Article parsing, metadata extraction | Maintenance status varies; test carefully |
| **Crawl4AI** | [github.com/unclecode/crawl4ai](https://github.com/unclecode/crawl4ai) | LLM-oriented crawler/extractor | Free/self-hosted | Markdown extraction/crawling for AI apps | Useful as a self-hosted alternative to hosted extraction APIs |

---

# 6. Excluded or not in the main table

These are relevant providers, but I did **not** include them in the main list because their free usage appears to be one-time, trial-based, expiring, or not clearly recurring.

| Provider | Why excluded from main recurring-free list | Relevant note |
|---|---|---|
| **Serper** | The visible free offer is **2,500 free queries**, but the page does not clearly state this renews monthly/daily. Purchased credits are valid for 6 months. | Very useful and cheap Google SERP API; still worth testing, but not counted as continual-free. |
| **SearchAPI.io** | Pricing page shows **100 free requests**, but not clearly recurring. | Broad SERP/API coverage; verify account behavior before relying on it as a free tier. |
| **Linkup** | Pricing page says **4,000 queries for free**, but does not clearly state recurring monthly/daily renewal. Startup credits are separate and one-time. | Strong AI-agent search/fetch/research API; include in paid evaluation. |
| **Crawlbase** | Page shows **1,000 free requests** and a **30-day Smart AI Proxy trial**, but recurring free usage was not clearly confirmed. | Useful scraping/proxy/crawling vendor; treat as trial unless confirmed otherwise. |
| **Zyte** | Free usage appeared to be a **$5 / 30-day credit**, not continual. | Strong managed scraping stack, but not a perpetual free tier. |
| **Scrapingdog** | Free offer appeared to be **1,000 credits / 30-day trial**, not continual. | Good breadth, but not counted here. |
| **Bright Data** | Primarily free trial / contact / enterprise-style pricing; no clear evergreen free quota found. | Important enterprise option for SERP, browser, unlocker, proxies, and MCP, but not free-tier core. |
| **DataForSEO** | Pricing indicates a minimum payment / paid usage model; no clear perpetual free tier found. | Extremely broad SEO/SERP/data API suite; good paid candidate. |
| **ValueSERP / Traject Data** | Free trial/PAYG, not clearly recurring-free. | Good paid SERP alternative. |
| **Kagi Search API** | Closed beta / paid pricing, no free tier found. | Interesting premium search API, but not a free-tier candidate. |

---

# 7. Suggested free-tier research-agent routing

A practical free-tier-first architecture could look like this:

## Discovery

- **General web:** Exa, Tavily, Brave Search API, SerpApi, serpstack, Jina Search.
- **News:** Currents, GNews, NewsAPI, GDELT, Guardian, Brave News, SerpApi Google News.
- **Scholarly:** OpenAlex, Semantic Scholar, Crossref, arXiv, PubMed/NCBI, DataCite, OpenCitations, ORCID.
- **Blogs/platforms:** WordPress REST, Blogger, Forem/DEV, HN API, Reddit API.

## Extraction

- **First pass:** Jina Reader.
- **If JS/rendering needed:** ScrapingAnt, Firecrawl, Browserless.
- **If structured extraction needed:** Diffbot.
- **If marketplace crawler needed:** Apify.

## Corpora / offline intelligence

- **Historical web:** Internet Archive, Common Crawl.
- **Global news/events:** GDELT.
- **Entity enrichment:** Wikidata, OpenAlex, ORCID, Crossref, DataCite.

## Cache fields worth storing

| Field | Why |
|---|---|
| `url` | Original URL discovered |
| `canonical_url` | Deduplication |
| `source_domain` | Reputation/source grouping |
| `title` | Display/search |
| `author` | Provenance |
| `published_at` | Temporal reasoning |
| `retrieved_at` | Audit trail |
| `discovery_provider` | Which API found it |
| `extraction_provider` | Which API extracted it |
| `raw_metadata` | Preserve provider-specific fields |
| `markdown_text` | LLM-ready content |
| `html_snapshot_ref` | Optional replay/debug |
| `license_or_terms_flag` | Compliance routing |
| `robots_policy_snapshot` | Crawl compliance |
| `content_hash` | Deduplication |
| `embedding_id` | Retrieval system integration |
| `citation_ids` | Crossref/OpenAlex/Semantic Scholar links |
| `confidence_score` | Extraction quality/provenance quality |
| `paywall_or_access_status` | Avoid broken full-text assumptions |

---

# 8. Highest-leverage free-tier stack

If you want the highest immediate leverage without paying, start with:

| Layer | Providers |
|---|---|
| Web search | Exa + Tavily + Brave Search + SerpApi + serpstack |
| News discovery | Currents + GNews + NewsAPI + GDELT + Guardian |
| Article/blog extraction | Jina Reader + Firecrawl + ScrapingAnt |
| Hard pages / JS rendering | Browserless + ScrapingAnt + Apify |
| Scholarly discovery | OpenAlex + Semantic Scholar + Crossref + PubMed/NCBI + arXiv |
| Citation graph | OpenCitations + Crossref + Semantic Scholar + OpenAlex |
| Historical web/news | Internet Archive + Common Crawl + GDELT |
| Entity graph | Wikidata + ORCID + OpenAlex |
| Platform-specific blogs/discussions | WordPress REST + Blogger + Forem/DEV + Hacker News + Reddit |

---

# 9. Strategic tool primitives for a deep research agent

The provider list above should be treated as implementation inventory, not as the agent-facing API. A deep research agent needs stable primitives that survive provider churn, quota exhaustion, and uneven result quality.

## Core primitives

| Primitive | Purpose | Typical providers | Primary output |
|---|---|---|---|
| `search.web` | Broad web discovery for a query | Exa, Tavily, Brave, SerpApi, serpstack, Jina Search | Ranked `SearchResult[]` |
| `search.news` | Recent or historical news discovery | GDELT, Currents, GNews, NewsAPI, Guardian, Brave News | Ranked `SearchResult[]` with freshness metadata |
| `search.scholar` | Academic paper and citation discovery | OpenAlex, Semantic Scholar, Crossref, PubMed, arXiv | `WorkResult[]` |
| `search.platform` | Source-specific discovery | WordPress, Blogger, Forem, HN, Reddit | `PlatformResult[]` |
| `fetch.url` | Fetch raw URL content without heavy extraction | Native HTTP, Jina Reader, Firecrawl, ScrapingAnt | `FetchedDocument` |
| `extract.article` | Convert a URL or HTML page to readable text/markdown | Jina Reader, Firecrawl, Diffbot, trafilatura, Readability.js | `ExtractedDocument` |
| `render.browser` | Load JS-heavy pages, screenshots, DOM snapshots | Browserless, ScrapingAnt, Playwright, Apify | `RenderedPage` |
| `crawl.site` | Traverse many URLs from one site | Firecrawl, Apify, Scrapy, Crawlee, Crawl4AI | `CrawlResult[]` |
| `resolve.identity` | Normalize entities, authors, orgs, papers, DOIs | Wikidata, ORCID, OpenAlex, Crossref, DataCite | `EntityRecord` |
| `enrich.citations` | Expand references/citations/open-access copies | OpenAlex, Semantic Scholar, Crossref, OpenCitations, Unpaywall | `CitationGraph` |
| `archive.lookup` | Find historical captures or corpus records | Internet Archive, Common Crawl, GDELT | `ArchiveRecord[]` |
| `cache.get/put` | Reuse prior results and enforce provenance | Local SQLite, object store, content-addressed files | Cached normalized records |

## Non-core but important primitives

| Primitive | Why it matters |
|---|---|
| `classify.source` | Estimate whether a source is primary, secondary, spammy, official, user-generated, scholarly, or commercial. |
| `detect.paywall` | Avoid wasting extraction quota on pages unlikely to expose useful text. |
| `dedupe.results` | Merge the same article/paper discovered from multiple providers. |
| `rank.evidence` | Prefer source diversity, authority, recency, primary-source proximity, and citation support over provider rank alone. |
| `check.compliance` | Apply robots, terms, license, user-agent, retention, and commercial-use flags before crawling or storing content. |
| `budget.estimate` | Predict quota and cost before launching broad searches, crawls, or rendered fetches. |

The agent should call these primitives, not direct provider names. Provider selection should be a router decision based on task type, freshness requirements, cost budget, API keys available, quota remaining, and prior observed quality.

---

# 10. Recommended abstraction model

Use a small set of Go interfaces, then attach provider adapters behind them. Keep the interfaces capability-oriented rather than provider-oriented.

## Provider capability interfaces

```go
type SearchProvider interface {
    Name() string
    Capabilities() ProviderCapabilities
    Search(ctx context.Context, req SearchRequest) (*SearchResponse, error)
}

type ExtractProvider interface {
    Name() string
    Capabilities() ProviderCapabilities
    Extract(ctx context.Context, req ExtractRequest) (*ExtractResponse, error)
}

type RenderProvider interface {
    Name() string
    Capabilities() ProviderCapabilities
    Render(ctx context.Context, req RenderRequest) (*RenderResponse, error)
}

type CorpusProvider interface {
    Name() string
    Capabilities() ProviderCapabilities
    Query(ctx context.Context, req CorpusRequest) (*CorpusResponse, error)
}
```

## Capability metadata

Each provider adapter should declare metadata the router can reason over:

| Field | Example values | Use |
|---|---|---|
| `domains` | `web`, `news`, `scholar`, `social`, `blogs`, `archive` | Match provider to task. |
| `freshness` | `realtime`, `delayed`, `historical`, `static` | Avoid stale providers for breaking-news tasks. |
| `content_support` | `url`, `html`, `markdown`, `pdf`, `image`, `screenshot`, `metadata_only` | Select fetch/extraction path. |
| `auth_required` | `none`, `api_key`, `oauth`, `custom` | Drive setup checks. |
| `rate_model` | `rpm`, `rps`, `daily`, `monthly`, `credits`, `shared_pool` | Enforce budgets. |
| `quota_scope` | `key`, `account`, `ip`, `provider_global`, `unknown` | Avoid bad assumptions. |
| `commercial_flag` | `allowed`, `restricted`, `unknown`, `custom_terms` | Compliance routing. |
| `best_for` | `general_recall`, `scholarly_graph`, `article_text`, `js_pages` | Improve default routing. |
| `failure_modes` | `captcha`, `403`, `quota`, `partial_text`, `delayed_news` | Trigger targeted fallback. |

## Normalized records

Provider responses should be normalized immediately, while preserving raw payloads for audit/debug.

```go
type SearchResult struct {
    URL              string
    CanonicalURL     string
    Title            string
    Snippet          string
    PublishedAt      *time.Time
    SourceName       string
    SourceDomain     string
    Provider         string
    ProviderRank     int
    Score            float64
    ResultType       string // webpage, news, paper, discussion, dataset, video, image
    Language         string
    Raw              json.RawMessage
}

type ExtractedDocument struct {
    URL              string
    CanonicalURL     string
    Title            string
    Author           string
    PublishedAt      *time.Time
    RetrievedAt      time.Time
    Markdown         string
    PlainText        string
    HTMLRef          string
    Provider         string
    ExtractionMethod string
    ContentHash      string
    QualityScore     float64
    Compliance       ComplianceRecord
    Raw              json.RawMessage
}
```

---

# 11. Router and fallback strategy

The router should implement intent-aware fanout and fallback rather than a fixed provider order.

## Discovery routing

| Research intent | Primary route | Fallback route | Notes |
|---|---|---|---|
| Broad factual web research | Exa/Tavily + Brave | SerpApi/serpstack + Jina Search | Blend AI-native and conventional search for diversity. |
| Recent news | GDELT + Brave News + Currents | GNews/NewsAPI/Guardian/SerpApi News | Model provider delay explicitly. |
| Official/source-of-truth lookup | Brave/SerpApi with domain filters | Jina Search + direct site search APIs | Prefer official domains and known repositories. |
| Scholarly survey | OpenAlex + Semantic Scholar | Crossref + PubMed/arXiv/DataCite | Use citation graph expansion after initial discovery. |
| Biomedical topic | PubMed/NCBI + Europe PMC | Semantic Scholar + OpenAlex | Domain-specific APIs beat generic web search. |
| DOI or paper enrichment | Crossref + OpenAlex | Semantic Scholar + DataCite + Unpaywall | Normalize DOI and external IDs early. |
| Developer/community signals | HN + Reddit + Forem | Brave/SerpApi restricted to relevant domains | Treat discussions as context, not primary evidence. |
| Historical web/news | GDELT + Internet Archive | Common Crawl URL index | Good for timelines and deleted/changed pages. |

## Extraction routing

| Page condition | Primary route | Fallback route |
|---|---|---|
| Normal article/blog/docs page | Jina Reader | Local fetch + Readability/trafilatura, then Firecrawl |
| PDF | Jina Reader if supported | Direct download + local PDF text extraction |
| JS-heavy page | Browserless or ScrapingAnt | Playwright local, then Apify actor |
| Anti-bot or blocked page | Firecrawl/ScrapingAnt | Browserless with conservative settings |
| Structured article/product/entity extraction | Diffbot | Firecrawl extract, then local parser |
| Multi-page site exploration | Firecrawl map/crawl | Apify actor, Crawlee/Scrapy self-hosted |

## Fallback triggers

Fallback should be based on observable failure classes:

| Trigger | Router response |
|---|---|
| `quota_exceeded` | Disable provider until reset time; retry equivalent provider. |
| `rate_limited` | Backoff and queue if latency allows; otherwise choose another provider. |
| `auth_missing` | Skip provider and return setup hint in CLI diagnostics. |
| `partial_content` | Try stronger extraction/rendering provider. |
| `low_quality_text` | Try local extractor or rendered DOM path. |
| `blocked_or_captcha` | Escalate to browser/proxy-capable provider only if compliance policy allows. |
| `stale_results` | Route to realtime/news provider. |
| `metadata_only` | Add a fetch/extract stage before presenting as evidence. |
| `terms_restricted` | Mark as unavailable for current policy profile. |

---

# 12. Rate-limit and quota budgeting

Free-tier-first routing only works if quota management is explicit and local.

## Required quota model

Each provider should have a local quota record:

```yaml
providers:
  brave:
    enabled: true
    auth_env: BRAVE_API_KEY
    limits:
      monthly_credits: 5.00
      request_cost:
        search_web: 0.005
        search_news: 0.005
      rpm: 60
    reset:
      cadence: monthly
      timezone: UTC
  jina_reader:
    enabled: true
    auth_env: JINA_API_KEY
    limits:
      rpm: 500
      scope: key
    reset:
      cadence: rolling_minute
```

## Budget rules

| Rule | Reason |
|---|---|
| Track estimated and observed usage separately. | Some providers charge by credits, tokens, browser time, or endpoint-specific units. |
| Reserve quota for high-value tasks. | Do not burn all free SERP quota on exploratory fanout. |
| Use cheap metadata providers before expensive extraction. | Search result snippets and paper metadata often remove bad candidates before fetch. |
| Cache every successful normalized response. | Free quotas are too small to waste on repeated calls. |
| Cache negative outcomes briefly. | Avoid retry storms on blocked pages, missing records, and quota errors. |
| Apply per-provider concurrency caps. | Monthly free quota is not the only limit; many providers also cap RPS/RPM/concurrency. |
| Return budget diagnostics in CLI output. | Users need to know why one provider was skipped or delayed. |

## Suggested quota states

| State | Meaning | Router behavior |
|---|---|---|
| `healthy` | Plenty of quota and recent success | Eligible for normal routing. |
| `conserve` | Below reserve threshold | Use only for high-confidence matches. |
| `exhausted` | No usable quota until reset | Skip unless user explicitly overrides. |
| `cooldown` | Short-term rate limit hit | Retry later; use fallback now. |
| `degraded` | Repeated errors or poor quality | Lower priority until health check passes. |
| `disabled` | Missing key, terms mismatch, or user disabled | Never route automatically. |

---

# 13. Go CLI design

The CLI should expose research primitives, not provider-specific commands first. Provider-specific commands are still useful for diagnostics and setup.

## Proposed command surface

```text
forage search web "query" --limit 20 --freshness month --sources diverse
forage search news "query" --since 2026-05-01 --limit 50
forage search scholar "query" --field cs --since 2023
forage fetch https://example.com/article --format markdown
forage extract urls.txt --out docs.jsonl --fallback render
forage crawl https://example.com --max-pages 100 --policy polite
forage enrich doi 10.1234/example --citations --open-access
forage archive lookup https://example.com/article
forage providers list
forage providers doctor
forage providers quota
forage config init
```

## Useful global flags

| Flag | Purpose |
|---|---|
| `--profile free` | Use only recurring-free or local/self-hosted providers. |
| `--profile paid-ok` | Allow paid providers within configured budget. |
| `--policy strict` | Enforce conservative compliance and robots behavior. |
| `--policy research` | Allow broader metadata/corpus querying, still respecting provider terms. |
| `--max-cost 0` | Hard no-paid-spend mode. |
| `--max-requests N` | Stop runaway fanout. |
| `--providers a,b,c` | Pin routing for reproducibility. |
| `--exclude-provider x` | Disable a provider for one command. |
| `--cache only` | Do not call network providers. |
| `--cache refresh` | Prefer cache but refresh stale records. |
| `--json` | Machine-readable output for agents. |
| `--jsonl` | Streamable output for long runs. |
| `--explain-routing` | Show why providers were selected/skipped. |

## Output modes

| Mode | Use |
|---|---|
| Human table | Manual exploration. |
| JSON | Agent tool calls and structured automation. |
| JSONL | Batch extraction/crawling. |
| Markdown bundle | Direct research notes with citations. |
| SQLite cache | Local reusable corpus and provenance store. |

## Package layout

```text
/cmd/forage
  main.go
/internal/config
  config.go
  credentials.go
/internal/router
  router.go
  policy.go
  budget.go
/internal/providers
  brave/
  exa/
  tavily/
  jina/
  firecrawl/
  openalex/
  crossref/
  pubmed/
  gdelt/
  ...
/internal/normalize
  search.go
  document.go
  scholar.go
/internal/cache
  sqlite.go
  blobs.go
/internal/compliance
  robots.go
  terms.go
  policy.go
/internal/quality
  extraction.go
  ranking.go
/pkg/forage
  public SDK types and interfaces
```

Keep the public SDK small. Most provider-specific details belong under `internal/providers` until there is a real need to expose them.

---

# 14. Configuration and setup experience

The setup path should make partial configuration useful. A user should be able to run the CLI with only local tools, then progressively add API keys.

## Configuration sources

Recommended precedence:

1. CLI flags.
2. Environment variables.
3. Project config file, for example `.forage.yaml`.
4. User config file, for example `$HOME/.config/forage/config.yaml`.
5. Built-in defaults.

## Provider setup groups

| Setup group | Providers | Why |
|---|---|---|
| `no-key` | HN, GDELT, Crossref, arXiv, Wikidata, Common Crawl, Internet Archive, public WordPress REST where available | Lets the CLI work immediately. |
| `easy-key` | Brave, Jina, OpenAlex if needed, NCBI, SerpApi, Tavily, Exa, Currents, GNews | API key only; good first guided setup. |
| `oauth` | Reddit, WordPress.com, ORCID public credentials where needed | Requires more setup and clearer docs. |
| `browser` | Browserless, ScrapingAnt, Apify, local Playwright | Needed for hard pages and rendering. |
| `local` | Readability.js, trafilatura, Playwright, Scrapy/Crawlee/Crawl4AI | Reduces external spend; needs runtime dependencies. |

## `providers doctor`

The doctor command should check:

| Check | Example output |
|---|---|
| API key presence | `brave: configured via BRAVE_API_KEY` |
| Minimal request works | `jina_reader: ok, 312 ms` |
| Quota headers parsed | `serpapi: 248 monthly searches remaining` |
| Policy compatibility | `guardian: disabled under commercial profile without custom terms` |
| Cache database writable | `cache: ok, D:/.../forage/cache.sqlite` |
| Local tools available | `playwright: installed`, `trafilatura: missing` |

---

# 15. Evidence pipeline for agent use

A deep research agent should not treat search results as evidence. It should move through a staged evidence pipeline.

## Pipeline

1. **Clarify intent:** determine whether the task needs current web, news, scholarly literature, source-specific data, historical archives, or broad context.
2. **Plan budget:** choose fanout depth, provider mix, max requests, and reserve thresholds.
3. **Discover:** query multiple discovery providers appropriate to the intent.
4. **Normalize and dedupe:** canonicalize URLs/DOIs, merge duplicates, preserve provider provenance.
5. **Rank candidates:** prioritize primary sources, source diversity, recency, authority, citation graph signals, and likely extractability.
6. **Fetch/extract:** fetch only the top candidates first, escalating to rendered or structured extraction when necessary.
7. **Assess quality:** score extracted text for length, boilerplate ratio, title match, date confidence, language, and visible failure markers.
8. **Enrich:** add citations, archived versions, entity IDs, author/org metadata, and related works when useful.
9. **Cache:** store normalized results, raw payload refs, content hashes, and policy decisions.
10. **Return evidence pack:** provide documents with provenance, timestamps, extraction methods, and confidence indicators.

## Evidence pack schema

```json
{
  "query": "example research question",
  "generated_at": "2026-05-22T00:00:00Z",
  "routing": {
    "profile": "free",
    "providers_used": ["brave", "exa", "jina_reader"],
    "providers_skipped": [
      {"provider": "serpapi", "reason": "quota_conserve"}
    ]
  },
  "results": [
    {
      "title": "Example",
      "url": "https://example.com",
      "canonical_url": "https://example.com",
      "source_domain": "example.com",
      "published_at": "2026-05-01T00:00:00Z",
      "retrieved_at": "2026-05-22T00:00:00Z",
      "discovery_provider": "brave",
      "extraction_provider": "jina_reader",
      "content_hash": "sha256:...",
      "quality_score": 0.91,
      "compliance_flags": [],
      "markdown": "..."
    }
  ]
}
```

---

# 16. Implementation roadmap

## Phase 1: Useful local CLI

- Build `forage config init`, `providers list`, `providers doctor`, and a local SQLite cache.
- Implement no-key providers first: HN, GDELT, Crossref, arXiv, Wikidata, Internet Archive metadata, public WordPress REST probing.
- Implement local fetch plus local extraction using a simple HTTP client and a local readability/trafilatura path.
- Add normalized `SearchResult`, `ExtractedDocument`, and `ProviderStatus` types.

## Phase 2: Free-tier search and extraction router

- Add Brave, Jina Reader/Search, Tavily, Exa, SerpApi, Firecrawl, and OpenAlex adapters.
- Add budget manager with quota states, cooldowns, reserve thresholds, and `--explain-routing`.
- Implement `forage search web`, `forage search news`, `forage fetch`, and `forage extract`.
- Add result deduplication by canonical URL, normalized title, DOI, and content hash.

## Phase 3: Scholarly and citation workflows

- Add Semantic Scholar, PubMed/NCBI, DataCite, OpenCitations, Unpaywall, DOAJ, and Europe PMC where available.
- Implement `forage search scholar`, `forage enrich doi`, and citation graph expansion.
- Add paper/work normalization around DOI, PMID, arXiv ID, OpenAlex ID, Semantic Scholar ID, and ORCID.

## Phase 4: Hard-page and crawl support

- Add Browserless, ScrapingAnt, Apify, and local Playwright rendering.
- Implement `forage render`, `forage crawl`, crawl budgets, robots handling, and crawl frontier persistence.
- Add extraction quality scoring and automatic escalation from normal fetch to render.

## Phase 5: Agent-ready research packs

- Implement `forage research "question"` as an orchestration command that emits an evidence pack.
- Add source classification, rank fusion, provenance summaries, and cache-backed replay.
- Add policy profiles for personal research, commercial use, strict robots compliance, and offline/cache-only operation.

---

# 17. Design principles

1. **Expose primitives, hide providers.** Agents should ask for search, extraction, rendering, enrichment, and archives; the router chooses providers.
2. **Make every result auditable.** Store provider name, raw payload refs, retrieval time, extraction method, cache status, and policy flags.
3. **Prefer cheap discovery before expensive extraction.** Search and metadata should prune candidate URLs before fetching or rendering.
4. **Use fallback for specific failure classes.** A blocked JS page needs a different fallback than stale news or quota exhaustion.
5. **Treat rate limits as product behavior.** Quota state should be visible, testable, cached, and part of routing.
6. **Design for partial setup.** The CLI should work with no keys, improve with a few easy keys, and scale into paid providers later.
7. **Keep compliance explicit.** Provider terms, commercial-use restrictions, robots policy, and retention rules should be first-class routing inputs.
8. **Normalize early, preserve raw data.** The agent needs consistent records, but engineers need raw provider responses for debugging.
9. **Separate discovery from evidence.** Search results are candidates; extracted, timestamped, provenance-rich documents are evidence.
10. **Make reproducibility possible.** Pin providers, cache records, emit routing explanations, and support cache-only replay.
