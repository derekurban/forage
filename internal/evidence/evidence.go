package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type Pack struct {
	SchemaVersion  string   `json:"schema_version"`
	ID             string   `json:"id"`
	GeneratedAt    string   `json:"generated_at"`
	Query          string   `json:"query,omitempty"`
	Policy         string   `json:"policy"`
	Records        []Record `json:"records"`
	Compliance     []string `json:"compliance_flags,omitempty"`
	RawPayloadRefs []string `json:"raw_payload_refs,omitempty"`
}

type Record struct {
	Provider         string `json:"provider,omitempty"`
	URL              string `json:"url,omitempty"`
	Title            string `json:"title,omitempty"`
	RetrievedAt      string `json:"retrieved_at,omitempty"`
	ContentHash      string `json:"content_hash"`
	ExtractionMethod string `json:"extraction_method,omitempty"`
	CacheStatus      string `json:"cache_status,omitempty"`
	Payload          any    `json:"payload"`
}

func Create(dir string, query string, policy string, items []any) (string, Pack, error) {
	p := Pack{SchemaVersion: "forage.evidence.v1", GeneratedAt: time.Now().UTC().Format(time.RFC3339), Query: query, Policy: policy}
	for _, item := range items {
		p.Records = append(p.Records, normalizeRecord(item))
	}
	sum := sha256.Sum256([]byte(p.GeneratedAt + query))
	p.ID = hex.EncodeToString(sum[:8])
	if err := os.MkdirAll(filepath.Join(dir, "evidence"), 0o755); err != nil {
		return "", p, err
	}
	path := filepath.Join(dir, "evidence", p.ID+".json")
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return "", p, err
	}
	return path, p, os.WriteFile(path, b, 0o600)
}

func normalizeRecord(item any) Record {
	b, _ := json.Marshal(item)
	rec := Record{Payload: item, ContentHash: hashBytes(b)}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return rec
	}
	copyString := func(key string) string {
		if v, ok := m[key].(string); ok {
			return v
		}
		return ""
	}
	rec.Provider = copyString("provider")
	if rec.Provider == "" {
		rec.Provider = copyString("fetch_provider")
	}
	if rec.Provider == "" {
		rec.Provider = copyString("search_provider")
	}
	rec.URL = copyString("url")
	rec.Title = copyString("title")
	rec.RetrievedAt = copyString("retrieved_at")
	rec.ExtractionMethod = copyString("extraction_method")
	rec.CacheStatus = copyString("cache_status")
	if rec.Provider == "" {
		if doc, ok := m["document"].(map[string]any); ok {
			if v, ok := doc["provider"].(string); ok {
				rec.Provider = v
			}
			if v, ok := doc["url"].(string); ok {
				rec.URL = v
			}
			if v, ok := doc["title"].(string); ok {
				rec.Title = v
			}
			if v, ok := doc["retrieved_at"].(string); ok {
				rec.RetrievedAt = v
			}
			if v, ok := doc["extraction_method"].(string); ok {
				rec.ExtractionMethod = v
			}
			if v, ok := doc["content_hash"].(string); ok && v != "" {
				rec.ContentHash = v
			}
		}
	}
	return rec
}

func hashBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func Inspect(path string) (Pack, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Pack{}, err
	}
	var p Pack
	return p, json.Unmarshal(b, &p)
}
