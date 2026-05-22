package state

import (
	"path/filepath"
	"testing"
	"time"
)

func TestProviderStateRoundTrip(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	remaining := int64(42)
	limit := int64(100)
	used := int64(58)
	httpStatus := 429
	err = st.UpsertProviderState(ProviderState{
		Provider:       "brave",
		Status:         "cooldown",
		Reason:         "rate_limited",
		Limit:          &limit,
		Remaining:      &remaining,
		Used:           &used,
		RetryAfter:     "60",
		LastHTTPStatus: &httpStatus,
		Observed:       "retry-after=60",
	})
	if err != nil {
		t.Fatal(err)
	}
	states, err := st.ProviderStates()
	if err != nil {
		t.Fatal(err)
	}
	if len(states) != 1 {
		t.Fatalf("len(states) = %d", len(states))
	}
	got := states[0]
	if got.Provider != "brave" || got.Status != "cooldown" || got.Reason != "rate_limited" {
		t.Fatalf("unexpected state: %+v", got)
	}
	if got.Remaining == nil || *got.Remaining != 42 {
		t.Fatalf("remaining = %+v", got.Remaining)
	}
	if got.Limit == nil || *got.Limit != 100 {
		t.Fatalf("limit = %+v", got.Limit)
	}
	if got.Used == nil || *got.Used != 58 {
		t.Fatalf("used = %+v", got.Used)
	}
	if got.LastHTTPStatus == nil || *got.LastHTTPStatus != 429 {
		t.Fatalf("status = %+v", got.LastHTTPStatus)
	}
}

func TestNegativeCacheActive(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.PutNegativeCache("k", "cache_miss", 5*time.Second); err != nil {
		t.Fatal(err)
	}
	reason, ok, err := st.NegativeCacheActive("k")
	if err != nil {
		t.Fatal(err)
	}
	if !ok || reason != "cache_miss" {
		t.Fatalf("negative cache = %q %v", reason, ok)
	}
}

func TestPutRecordClearsNegativeCache(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.PutNegativeCache("k", "cache_miss", 5*time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := st.PutRecord("search", "k", "brave", "", "title", map[string]string{"ok": "true"}); err != nil {
		t.Fatal(err)
	}
	_, ok, err := st.NegativeCacheActive("k")
	if err != nil || ok {
		t.Fatalf("negative cache after PutRecord ok=%v err=%v", ok, err)
	}
}

func TestStats(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	stats, err := st.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if stats["provider_state_count"] != 0 {
		t.Fatalf("provider_state_count = %v", stats["provider_state_count"])
	}
}

func TestProviderUsageAndReset(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.IncrementProviderUsage("brave", 24*time.Hour); err != nil {
		t.Fatal(err)
	}
	usage, ok, err := st.ProviderUsage("brave")
	if err != nil || !ok {
		t.Fatalf("usage = %+v/%v/%v", usage, ok, err)
	}
	if usage.RequestCount != 1 {
		t.Fatalf("count = %d", usage.RequestCount)
	}
	if err := st.ResetProviderState("brave"); err != nil {
		t.Fatal(err)
	}
	_, ok, err = st.ProviderUsage("brave")
	if err != nil || ok {
		t.Fatalf("usage after reset ok=%v err=%v", ok, err)
	}
}
