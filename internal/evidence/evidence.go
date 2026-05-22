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
	ID          string   `json:"id"`
	GeneratedAt string   `json:"generated_at"`
	Query       string   `json:"query,omitempty"`
	Policy      string   `json:"policy"`
	Items       []any    `json:"items"`
	Compliance  []string `json:"compliance_flags,omitempty"`
}

func Create(dir string, query string, policy string, items []any) (string, Pack, error) {
	p := Pack{GeneratedAt: time.Now().UTC().Format(time.RFC3339), Query: query, Policy: policy, Items: items}
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

func Inspect(path string) (Pack, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Pack{}, err
	}
	var p Pack
	return p, json.Unmarshal(b, &p)
}
