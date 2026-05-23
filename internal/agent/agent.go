package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/derekurban/forage/internal/apperr"
	"github.com/derekurban/forage/internal/capability"
	"github.com/derekurban/forage/internal/config"
	"github.com/derekurban/forage/internal/evidence"
	"github.com/derekurban/forage/internal/router"
)

const (
	KindAuto   = "auto"
	KindURL    = "url"
	KindDOI    = "doi"
	KindPaper  = "paper"
	KindAuthor = "author"
	KindQuery  = "query"
)

type Workflow struct {
	Router      router.Router
	Config      config.Config
	EvidenceDir string
}

type GatherRequest struct {
	Query          string `json:"query"`
	Mode           string `json:"mode"`
	Limit          int    `json:"limit"`
	Fetch          int    `json:"fetch"`
	CacheMode      string `json:"cache_mode"`
	MaxChars       int    `json:"max_chars"`
	SavePack       bool   `json:"save_pack"`
	ExplainRouting bool   `json:"explain_routing"`
}

type RetrieveRequest struct {
	Input          string `json:"input"`
	Kind           string `json:"kind"`
	CacheMode      string `json:"cache_mode"`
	MaxChars       int    `json:"max_chars"`
	SavePack       bool   `json:"save_pack"`
	ExplainRouting bool   `json:"explain_routing"`
}

type BriefRequest struct {
	Query          string `json:"query"`
	Mode           string `json:"mode"`
	Limit          int    `json:"limit"`
	Fetch          int    `json:"fetch"`
	Format         string `json:"format"`
	CacheMode      string `json:"cache_mode"`
	MaxChars       int    `json:"max_chars"`
	ExplainRouting bool   `json:"explain_routing"`
}

type Record struct {
	Title          string  `json:"title,omitempty"`
	URL            string  `json:"url,omitempty"`
	SourceDomain   string  `json:"source_domain,omitempty"`
	SearchProvider string  `json:"search_provider,omitempty"`
	FetchProvider  string  `json:"fetch_provider,omitempty"`
	Text           string  `json:"text,omitempty"`
	RetrievedAt    string  `json:"retrieved_at,omitempty"`
	ContentHash    string  `json:"content_hash,omitempty"`
	QualityScore   float64 `json:"quality_score,omitempty"`
	CacheStatus    string  `json:"cache_status,omitempty"`
}

type Routing struct {
	Search  []capability.RoutingDiagnostics `json:"search,omitempty"`
	Fetches []capability.RoutingDiagnostics `json:"fetches,omitempty"`
	Data    []capability.RoutingDiagnostics `json:"data,omitempty"`
}

type GatherResponse struct {
	Query            string   `json:"query"`
	Mode             string   `json:"mode"`
	Records          []Record `json:"records"`
	Routing          *Routing `json:"routing,omitempty"`
	CacheStatus      string   `json:"cache_status,omitempty"`
	EvidencePackPath string   `json:"evidence_pack_path,omitempty"`
}

type RetrieveResponse struct {
	Input            string   `json:"input"`
	Kind             string   `json:"kind"`
	Result           any      `json:"result,omitempty"`
	Records          []Record `json:"records,omitempty"`
	Routing          *Routing `json:"routing,omitempty"`
	EvidencePackPath string   `json:"evidence_pack_path,omitempty"`
}

type BriefResponse struct {
	Query   string   `json:"query"`
	Mode    string   `json:"mode"`
	Format  string   `json:"format"`
	Records []Record `json:"records"`
	Context string   `json:"context"`
	Routing *Routing `json:"routing,omitempty"`
}

func (w Workflow) Gather(ctx context.Context, req GatherRequest) (GatherResponse, *apperr.Error) {
	req = defaultGather(req)
	routing := &Routing{}
	var candidates []capability.SearchResult
	cacheStatus := ""
	for _, capName := range capabilitiesForMode(req.Mode) {
		resp, ae := w.Router.Search(ctx, capability.SearchRequest{Query: req.Query, Capability: capName, Limit: perCapabilityLimit(req.Limit, req.Mode), CacheMode: req.CacheMode, ExplainRouting: req.ExplainRouting})
		if ae != nil {
			continue
		}
		cacheStatus = mergeCacheStatus(cacheStatus, resp.CacheStatus)
		if req.ExplainRouting && resp.Routing != nil {
			routing.Search = append(routing.Search, *resp.Routing)
		}
		candidates = append(candidates, resp.Results...)
	}
	candidates = dedupeResults(candidates, req.Limit)
	if len(candidates) == 0 {
		return GatherResponse{}, apperr.New(apperr.CodeCapabilityExhausted, "No sources were found for gather.", apperr.ExitCapabilityExhausted)
	}
	records := w.fetchEvidence(ctx, candidates, req.Fetch, req.CacheMode, req.MaxChars, req.ExplainRouting, routing)
	if len(records) == 0 {
		return GatherResponse{}, apperr.New(apperr.CodeCapabilityExhausted, "No usable evidence records were gathered.", apperr.ExitCapabilityExhausted)
	}
	resp := GatherResponse{Query: req.Query, Mode: req.Mode, Records: records, CacheStatus: cacheStatus}
	if req.ExplainRouting {
		resp.Routing = routing
	}
	if req.SavePack {
		path, _, err := evidence.Create(w.evidenceDir(), req.Query, w.Config.Policy, recordsToAny(records))
		if err != nil {
			return GatherResponse{}, apperr.New(apperr.CodeGeneral, err.Error(), apperr.ExitGeneral)
		}
		resp.EvidencePackPath = path
	}
	return resp, nil
}

func (w Workflow) Retrieve(ctx context.Context, req RetrieveRequest) (RetrieveResponse, *apperr.Error) {
	req = defaultRetrieve(req)
	kind := req.Kind
	if kind == "" || kind == KindAuto {
		kind = ClassifyInput(req.Input)
	}
	routing := &Routing{}
	switch kind {
	case KindURL:
		resp, ae := w.Router.Fetch(ctx, capability.FetchRequest{URL: req.Input, CacheMode: req.CacheMode, ExplainRouting: req.ExplainRouting})
		if ae != nil {
			return RetrieveResponse{}, ae
		}
		if req.ExplainRouting && resp.Routing != nil {
			routing.Fetches = append(routing.Fetches, *resp.Routing)
		}
		record := recordFromDocument(resp.Document, capability.SearchResult{URL: req.Input}, req.MaxChars)
		record.CacheStatus = resp.CacheStatus
		return w.finishRetrieve(req, kind, resp, []Record{record}, routing)
	case KindDOI:
		id := normalizeDOI(req.Input)
		if id == "" {
			id = req.Input
		}
		resp, ae := w.Router.EnrichDOI(ctx, capability.DataRequest{ID: id, Query: id, CacheMode: req.CacheMode, ExplainRouting: req.ExplainRouting})
		if ae != nil {
			return RetrieveResponse{}, ae
		}
		if req.ExplainRouting && resp.Routing != nil {
			routing.Data = append(routing.Data, *resp.Routing)
		}
		return w.finishRetrieve(req, kind, resp.Data, []Record{recordFromData(id, resp.Data, req.MaxChars)}, routing)
	case KindAuthor:
		resp, ae := w.Router.EnrichAuthor(ctx, capability.DataRequest{ID: req.Input, Query: req.Input, Providers: []string{"orcid"}, CacheMode: req.CacheMode, ExplainRouting: req.ExplainRouting})
		if ae != nil {
			return RetrieveResponse{}, ae
		}
		if req.ExplainRouting && resp.Routing != nil {
			routing.Data = append(routing.Data, *resp.Routing)
		}
		return w.finishRetrieve(req, kind, resp.Data, []Record{recordFromData(req.Input, resp.Data, req.MaxChars)}, routing)
	case KindPaper:
		return w.retrievePaper(ctx, req, routing)
	default:
		gather, ae := w.Gather(ctx, GatherRequest{Query: req.Input, Mode: "auto", Limit: 8, Fetch: 5, CacheMode: req.CacheMode, MaxChars: req.MaxChars, SavePack: req.SavePack, ExplainRouting: req.ExplainRouting})
		if ae != nil {
			return RetrieveResponse{}, ae
		}
		return RetrieveResponse{Input: req.Input, Kind: KindQuery, Result: gather, Records: gather.Records, Routing: gather.Routing, EvidencePackPath: gather.EvidencePackPath}, nil
	}
}

func (w Workflow) Brief(ctx context.Context, req BriefRequest) (BriefResponse, *apperr.Error) {
	req = defaultBrief(req)
	gather, ae := w.Gather(ctx, GatherRequest{Query: req.Query, Mode: req.Mode, Limit: req.Limit, Fetch: req.Fetch, CacheMode: req.CacheMode, MaxChars: req.MaxChars, ExplainRouting: req.ExplainRouting})
	if ae != nil {
		return BriefResponse{}, ae
	}
	format := strings.ToLower(strings.TrimSpace(req.Format))
	if format == "" {
		format = "markdown"
	}
	contextText := RenderContext(req.Query, gather.Records, format)
	resp := BriefResponse{Query: req.Query, Mode: req.Mode, Format: format, Records: gather.Records, Context: contextText}
	if req.ExplainRouting {
		resp.Routing = gather.Routing
	}
	return resp, nil
}

func (w Workflow) retrievePaper(ctx context.Context, req RetrieveRequest, routing *Routing) (RetrieveResponse, *apperr.Error) {
	if isPMID(req.Input) {
		resp, ae := w.Router.Search(ctx, capability.SearchRequest{Query: req.Input, Capability: capability.SearchScholar, Limit: 1, Providers: []string{"pubmed"}, CacheMode: req.CacheMode, ExplainRouting: req.ExplainRouting})
		if ae != nil {
			return RetrieveResponse{}, ae
		}
		if req.ExplainRouting && resp.Routing != nil {
			routing.Search = append(routing.Search, *resp.Routing)
		}
		records := recordsFromSearch(resp.Results, req.MaxChars)
		return w.finishRetrieve(req, KindPaper, resp, records, routing)
	}
	providers := []string(nil)
	if isArxivID(req.Input) {
		providers = []string{"arxiv"}
	}
	resp, ae := w.Router.Search(ctx, capability.SearchRequest{Query: req.Input, Capability: capability.SearchScholar, Limit: 1, Providers: providers, CacheMode: req.CacheMode, ExplainRouting: req.ExplainRouting})
	if ae != nil {
		return RetrieveResponse{}, ae
	}
	if req.ExplainRouting && resp.Routing != nil {
		routing.Search = append(routing.Search, *resp.Routing)
	}
	records := recordsFromSearch(resp.Results, req.MaxChars)
	return w.finishRetrieve(req, KindPaper, resp, records, routing)
}

func (w Workflow) finishRetrieve(req RetrieveRequest, kind string, result any, records []Record, routing *Routing) (RetrieveResponse, *apperr.Error) {
	resp := RetrieveResponse{Input: req.Input, Kind: kind, Result: result, Records: records}
	if req.ExplainRouting {
		resp.Routing = routing
	}
	if req.SavePack {
		path, _, err := evidence.Create(w.evidenceDir(), req.Input, w.Config.Policy, recordsToAny(records))
		if err != nil {
			return RetrieveResponse{}, apperr.New(apperr.CodeGeneral, err.Error(), apperr.ExitGeneral)
		}
		resp.EvidencePackPath = path
	}
	return resp, nil
}

func (w Workflow) fetchEvidence(ctx context.Context, results []capability.SearchResult, fetchLimit int, cacheMode string, maxChars int, explain bool, routing *Routing) []Record {
	if fetchLimit <= 0 || fetchLimit > len(results) {
		fetchLimit = len(results)
	}
	records := make([]Record, 0, fetchLimit)
	for _, r := range results[:fetchLimit] {
		if strings.TrimSpace(r.URL) == "" {
			records = append(records, recordFromSearch(r, maxChars))
			continue
		}
		resp, ae := w.Router.Fetch(ctx, capability.FetchRequest{URL: r.URL, CacheMode: cacheMode, ExplainRouting: explain})
		if ae != nil {
			records = append(records, recordFromSearch(r, maxChars))
			continue
		}
		if explain && resp.Routing != nil {
			routing.Fetches = append(routing.Fetches, *resp.Routing)
		}
		rec := recordFromDocument(resp.Document, r, maxChars)
		rec.CacheStatus = resp.CacheStatus
		records = append(records, rec)
	}
	return records
}

func ClassifyInput(input string) string {
	s := strings.TrimSpace(input)
	lower := strings.ToLower(s)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		if strings.Contains(lower, "doi.org/") {
			return KindDOI
		}
		return KindURL
	}
	if normalizeDOI(s) != "" {
		return KindDOI
	}
	if isORCID(s) {
		return KindAuthor
	}
	if isArxivID(s) || isPMID(s) {
		return KindPaper
	}
	return KindQuery
}

func RenderContext(query string, records []Record, format string) string {
	var b strings.Builder
	if strings.EqualFold(format, "context") {
		b.WriteString("QUERY: " + query + "\n\n")
	} else {
		b.WriteString("# Forage Brief\n\n")
		b.WriteString("Query: " + query + "\n\n")
	}
	for i, r := range records {
		b.WriteString("SOURCE " + strconv.Itoa(i+1) + "\n")
		if r.Title != "" {
			b.WriteString("Title: " + r.Title + "\n")
		}
		if r.URL != "" {
			b.WriteString("URL: " + r.URL + "\n")
		}
		if r.RetrievedAt != "" {
			b.WriteString("Retrieved: " + r.RetrievedAt + "\n")
		}
		prov := strings.Trim(strings.Join([]string{r.SearchProvider, r.FetchProvider}, " "), " ")
		if prov != "" {
			b.WriteString("Providers: " + prov + "\n")
		}
		if r.Text != "" {
			b.WriteString("Excerpt:\n" + r.Text + "\n")
		}
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}

func defaultGather(req GatherRequest) GatherRequest {
	if req.Mode == "" {
		req.Mode = "auto"
	}
	if req.Limit <= 0 {
		req.Limit = 8
	}
	if req.Fetch <= 0 {
		req.Fetch = 5
	}
	if req.MaxChars <= 0 {
		req.MaxChars = 4000
	}
	if req.CacheMode == "" {
		req.CacheMode = "auto"
	}
	return req
}

func defaultRetrieve(req RetrieveRequest) RetrieveRequest {
	if req.Kind == "" {
		req.Kind = KindAuto
	}
	if req.MaxChars <= 0 {
		req.MaxChars = 4000
	}
	if req.CacheMode == "" {
		req.CacheMode = "auto"
	}
	return req
}

func defaultBrief(req BriefRequest) BriefRequest {
	if req.Mode == "" {
		req.Mode = "auto"
	}
	if req.Limit <= 0 {
		req.Limit = 6
	}
	if req.Fetch <= 0 {
		req.Fetch = 4
	}
	if req.MaxChars <= 0 {
		req.MaxChars = 1200
	}
	if req.Format == "" {
		req.Format = "markdown"
	}
	if req.CacheMode == "" {
		req.CacheMode = "auto"
	}
	return req
}

func capabilitiesForMode(mode string) []string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "news":
		return []string{capability.SearchNews}
	case "scholar":
		return []string{capability.SearchScholar}
	case "mixed":
		return []string{capability.SearchWeb, capability.SearchNews, capability.SearchScholar}
	default:
		return []string{capability.SearchWeb}
	}
}

func perCapabilityLimit(limit int, mode string) int {
	if strings.EqualFold(mode, "mixed") {
		if limit < 3 {
			return 1
		}
		return limit/3 + 1
	}
	return limit
}

func dedupeResults(in []capability.SearchResult, limit int) []capability.SearchResult {
	seen := map[string]bool{}
	var out []capability.SearchResult
	for _, r := range in {
		key := strings.ToLower(strings.TrimSpace(r.CanonicalURL))
		if key == "" {
			key = strings.ToLower(strings.TrimSpace(r.URL))
		}
		if key == "" {
			key = strings.ToLower(strings.TrimSpace(r.Title + "|" + r.SourceDomain))
		}
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, r)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

func recordFromDocument(doc capability.ExtractedDocument, search capability.SearchResult, maxChars int) Record {
	text := doc.Markdown
	if strings.TrimSpace(text) == "" {
		text = doc.PlainText
	}
	title := doc.Title
	if title == "" {
		title = search.Title
	}
	return Record{Title: title, URL: firstNonEmpty(doc.URL, search.URL), SourceDomain: firstNonEmpty(search.SourceDomain, domain(firstNonEmpty(doc.URL, search.URL))), SearchProvider: search.Provider, FetchProvider: doc.Provider, Text: trimText(text, maxChars), RetrievedAt: doc.RetrievedAt.Format(time.RFC3339), ContentHash: doc.ContentHash, QualityScore: doc.QualityScore}
}

func recordFromSearch(r capability.SearchResult, maxChars int) Record {
	return Record{Title: r.Title, URL: r.URL, SourceDomain: firstNonEmpty(r.SourceDomain, domain(r.URL)), SearchProvider: r.Provider, Text: trimText(r.Snippet, maxChars)}
}

func recordsFromSearch(results []capability.SearchResult, maxChars int) []Record {
	records := make([]Record, 0, len(results))
	for _, r := range results {
		records = append(records, recordFromSearch(r, maxChars))
	}
	return records
}

func recordFromData(id string, data any, maxChars int) Record {
	b, _ := json.Marshal(data)
	return Record{Title: id, Text: trimText(string(b), maxChars), ContentHash: hashBytes(b)}
}

func recordsToAny(records []Record) []any {
	items := make([]any, 0, len(records))
	for _, r := range records {
		items = append(items, r)
	}
	return items
}

func normalizeDOI(s string) string {
	s = strings.TrimSpace(s)
	lower := strings.ToLower(s)
	if strings.Contains(lower, "doi.org/") {
		if u, err := url.Parse(s); err == nil {
			s = strings.TrimPrefix(u.Path, "/")
		}
	}
	if regexp.MustCompile(`(?i)^10\.\d{4,9}/\S+$`).MatchString(s) {
		return s
	}
	return ""
}

func isORCID(s string) bool {
	return regexp.MustCompile(`^\d{4}-\d{4}-\d{4}-\d{3}[\dX]$`).MatchString(strings.TrimSpace(s))
}

func isArxivID(s string) bool {
	s = strings.TrimPrefix(strings.TrimSpace(strings.ToLower(s)), "arxiv:")
	return regexp.MustCompile(`^\d{4}\.\d{4,5}(v\d+)?$`).MatchString(s)
}

func isPMID(s string) bool {
	return regexp.MustCompile(`^\d{6,9}$`).MatchString(strings.TrimSpace(s))
}

func (w Workflow) evidenceDir() string {
	if w.EvidenceDir != "" {
		return w.EvidenceDir
	}
	return config.Dir()
}

func trimText(s string, max int) string {
	s = strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(s, " "))
	if max > 0 && len(s) > max {
		return strings.TrimSpace(s[:max]) + "..."
	}
	return s
}

func domain(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func mergeCacheStatus(a, b string) string {
	if a == "" {
		return b
	}
	if a == b || b == "" {
		return a
	}
	return "mixed"
}

func hashBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
