package capability

import "context"

type SearchProvider interface {
	ID() string
	Search(ctx context.Context, req SearchRequest) ([]SearchResult, error)
}

type FetchProvider interface {
	ID() string
	Fetch(ctx context.Context, req FetchRequest) (ExtractedDocument, error)
}

type ExtractProvider interface {
	ID() string
	Extract(ctx context.Context, req ExtractRequest) (ExtractedDocument, error)
}

type NewsProvider interface {
	ID() string
	SearchNews(ctx context.Context, req SearchRequest) ([]NewsResult, error)
}

type ScholarProvider interface {
	ID() string
	SearchScholar(ctx context.Context, req SearchRequest) ([]ScholarWork, error)
}

type ArchiveProvider interface {
	ID() string
	LookupArchive(ctx context.Context, url string, limit int) ([]ArchiveRecord, error)
}

type RenderProvider interface {
	ID() string
	Render(ctx context.Context, req FetchRequest) (ExtractedDocument, error)
}

type CrawlProvider interface {
	ID() string
	Crawl(ctx context.Context, url string, maxPages int) ([]ExtractedDocument, error)
}
