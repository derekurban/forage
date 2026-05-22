package state

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

type ProviderState struct {
	Provider       string `json:"provider"`
	Status         string `json:"status"`
	Reason         string `json:"reason,omitempty"`
	Remaining      *int64 `json:"remaining,omitempty"`
	ResetAt        string `json:"reset_at,omitempty"`
	RetryAfter     string `json:"retry_after,omitempty"`
	LastHTTPStatus *int   `json:"last_http_status,omitempty"`
	LastCheckedAt  string `json:"last_checked_at"`
	LastSuccessAt  string `json:"last_success_at,omitempty"`
	Observed       string `json:"observed,omitempty"`
}

type CacheRecord struct {
	Kind      string `json:"kind"`
	CacheKey  string `json:"cache_key"`
	Provider  string `json:"provider,omitempty"`
	URL       string `json:"url,omitempty"`
	Title     string `json:"title,omitempty"`
	Payload   string `json:"payload"`
	CreatedAt string `json:"created_at"`
}

type ProviderAttempt struct {
	ID         int64  `json:"id"`
	Capability string `json:"capability"`
	Provider   string `json:"provider"`
	Status     string `json:"status"`
	Reason     string `json:"reason,omitempty"`
	StartedAt  string `json:"started_at"`
	FinishedAt string `json:"finished_at,omitempty"`
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &Store{db: db}
	return s, s.migrate()
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
PRAGMA busy_timeout=5000;
PRAGMA journal_mode=WAL;
CREATE TABLE IF NOT EXISTS schema_migrations(version INTEGER PRIMARY KEY);
CREATE TABLE IF NOT EXISTS provider_state(
  provider TEXT PRIMARY KEY,
  status TEXT NOT NULL,
  reason TEXT,
  remaining INTEGER,
  reset_at TEXT,
  retry_after TEXT,
  last_http_status INTEGER,
  last_checked_at TEXT NOT NULL,
  last_success_at TEXT,
  observed TEXT
);
CREATE TABLE IF NOT EXISTS records(
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  kind TEXT NOT NULL,
  cache_key TEXT NOT NULL,
  provider TEXT,
  url TEXT,
  title TEXT,
  content_hash TEXT,
  payload TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_records_cache_key ON records(cache_key);
CREATE TABLE IF NOT EXISTS raw_payloads(
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  provider TEXT NOT NULL,
  kind TEXT NOT NULL,
  payload TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS negative_cache(
  cache_key TEXT PRIMARY KEY,
  reason TEXT NOT NULL,
  expires_at TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS provider_attempts(
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  capability TEXT NOT NULL,
  provider TEXT NOT NULL,
  status TEXT NOT NULL,
  reason TEXT,
  started_at TEXT NOT NULL,
  finished_at TEXT
);
INSERT OR IGNORE INTO schema_migrations(version) VALUES(1);
`)
	return err
}

func (s *Store) PutRecord(kind, cacheKey, provider, url, title string, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO records(kind,cache_key,provider,url,title,payload,created_at) VALUES(?,?,?,?,?,?,?)`,
		kind, cacheKey, provider, url, title, string(b), time.Now().UTC().Format(time.RFC3339))
	return err
}

func (s *Store) LatestRecord(kind, cacheKey string, ttl time.Duration, dest any) (bool, error) {
	var payload, created string
	err := s.db.QueryRow(`SELECT payload,created_at FROM records WHERE kind=? AND cache_key=? ORDER BY created_at DESC LIMIT 1`, kind, cacheKey).Scan(&payload, &created)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if ttl > 0 {
		t, err := time.Parse(time.RFC3339, created)
		if err == nil && time.Since(t) > ttl {
			return false, nil
		}
	}
	if err := json.Unmarshal([]byte(payload), dest); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Store) RecordAttempt(a ProviderAttempt) error {
	if a.StartedAt == "" {
		a.StartedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if a.FinishedAt == "" {
		a.FinishedAt = time.Now().UTC().Format(time.RFC3339)
	}
	_, err := s.db.Exec(`INSERT INTO provider_attempts(capability,provider,status,reason,started_at,finished_at) VALUES(?,?,?,?,?,?)`,
		a.Capability, a.Provider, a.Status, a.Reason, a.StartedAt, a.FinishedAt)
	return err
}

func (s *Store) PutRawPayload(provider, kind string, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO raw_payloads(provider,kind,payload,created_at) VALUES(?,?,?,?)`,
		provider, kind, string(b), time.Now().UTC().Format(time.RFC3339))
	return err
}

func (s *Store) PutNegativeCache(cacheKey, reason string, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	now := time.Now().UTC()
	_, err := s.db.Exec(`INSERT INTO negative_cache(cache_key,reason,expires_at,created_at) VALUES(?,?,?,?)
ON CONFLICT(cache_key) DO UPDATE SET reason=excluded.reason, expires_at=excluded.expires_at, created_at=excluded.created_at`,
		cacheKey, reason, now.Add(ttl).Format(time.RFC3339Nano), now.Format(time.RFC3339Nano))
	return err
}

func (s *Store) NegativeCacheActive(cacheKey string) (string, bool, error) {
	var reason, expires string
	err := s.db.QueryRow(`SELECT reason,expires_at FROM negative_cache WHERE cache_key=?`, cacheKey).Scan(&reason, &expires)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	t, err := time.Parse(time.RFC3339, expires)
	if err == nil && time.Now().UTC().After(t) {
		_, _ = s.db.Exec(`DELETE FROM negative_cache WHERE cache_key=?`, cacheKey)
		return "", false, nil
	}
	return reason, true, nil
}

func (s *Store) UpsertProviderState(ps ProviderState) error {
	if ps.LastCheckedAt == "" {
		ps.LastCheckedAt = time.Now().UTC().Format(time.RFC3339)
	}
	_, err := s.db.Exec(`
INSERT INTO provider_state(provider,status,reason,remaining,reset_at,retry_after,last_http_status,last_checked_at,last_success_at,observed)
VALUES(?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(provider) DO UPDATE SET
  status=excluded.status,
  reason=excluded.reason,
  remaining=excluded.remaining,
  reset_at=excluded.reset_at,
  retry_after=excluded.retry_after,
  last_http_status=excluded.last_http_status,
  last_checked_at=excluded.last_checked_at,
  last_success_at=excluded.last_success_at,
  observed=excluded.observed
`, ps.Provider, ps.Status, ps.Reason, ps.Remaining, ps.ResetAt, ps.RetryAfter, ps.LastHTTPStatus, ps.LastCheckedAt, ps.LastSuccessAt, ps.Observed)
	return err
}

func (s *Store) ProviderStates() ([]ProviderState, error) {
	rows, err := s.db.Query(`SELECT provider,status,COALESCE(reason,''),remaining,COALESCE(reset_at,''),COALESCE(retry_after,''),last_http_status,last_checked_at,COALESCE(last_success_at,''),COALESCE(observed,'') FROM provider_state ORDER BY provider`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ProviderState{}
	for rows.Next() {
		var ps ProviderState
		var remaining sql.NullInt64
		var status sql.NullInt64
		if err := rows.Scan(&ps.Provider, &ps.Status, &ps.Reason, &remaining, &ps.ResetAt, &ps.RetryAfter, &status, &ps.LastCheckedAt, &ps.LastSuccessAt, &ps.Observed); err != nil {
			return nil, err
		}
		if remaining.Valid {
			ps.Remaining = &remaining.Int64
		}
		if status.Valid {
			v := int(status.Int64)
			ps.LastHTTPStatus = &v
		}
		out = append(out, ps)
	}
	return out, rows.Err()
}

func (s *Store) ProviderState(provider string) (ProviderState, bool, error) {
	var ps ProviderState
	var remaining sql.NullInt64
	var status sql.NullInt64
	err := s.db.QueryRow(`SELECT provider,status,COALESCE(reason,''),remaining,COALESCE(reset_at,''),COALESCE(retry_after,''),last_http_status,last_checked_at,COALESCE(last_success_at,''),COALESCE(observed,'') FROM provider_state WHERE provider=?`, provider).Scan(&ps.Provider, &ps.Status, &ps.Reason, &remaining, &ps.ResetAt, &ps.RetryAfter, &status, &ps.LastCheckedAt, &ps.LastSuccessAt, &ps.Observed)
	if err == sql.ErrNoRows {
		return ProviderState{}, false, nil
	}
	if err != nil {
		return ProviderState{}, false, err
	}
	if remaining.Valid {
		ps.Remaining = &remaining.Int64
	}
	if status.Valid {
		v := int(status.Int64)
		ps.LastHTTPStatus = &v
	}
	return ps, true, nil
}

func (s *Store) ResetProviderState(provider string) error {
	_, err := s.db.Exec(`DELETE FROM provider_state WHERE provider=?`, provider)
	return err
}

func (s *Store) ClearCache() error {
	_, err := s.db.Exec(`DELETE FROM records; DELETE FROM raw_payloads; DELETE FROM negative_cache;`)
	return err
}

func (s *Store) Stats() (map[string]any, error) {
	stats := map[string]any{}
	var providerStates, records int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM provider_state`).Scan(&providerStates); err != nil {
		return nil, err
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM records`).Scan(&records); err != nil {
		return nil, err
	}
	stats["provider_state_count"] = providerStates
	stats["record_count"] = records
	var attempts, rawPayloads, negative int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM provider_attempts`).Scan(&attempts)
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM raw_payloads`).Scan(&rawPayloads)
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM negative_cache`).Scan(&negative)
	stats["provider_attempt_count"] = attempts
	stats["raw_payload_count"] = rawPayloads
	stats["negative_cache_count"] = negative
	return stats, nil
}
