package providers

import "testing"

func TestRegistryContainsRecurringFreeProductionSet(t *testing.T) {
	ps := Registry()
	if len(ps) < 12 {
		t.Fatalf("Registry() returned %d providers", len(ps))
	}
	for _, id := range []string{"brave", "jina", "tavily", "exa", "hackernews", "crossref", "arxiv"} {
		p, ok := ByID(id)
		if !ok {
			t.Fatalf("missing provider %s", id)
		}
		if p.Status != CapabilitySupported {
			t.Fatalf("%s status = %s", id, p.Status)
		}
	}
	p, ok := ByID("firecrawl")
	if !ok {
		t.Fatal("missing firecrawl")
	}
	if p.Status != CapabilitySupported {
		t.Fatalf("firecrawl status = %s", p.Status)
	}
}

func TestRegistryCredentialFieldsAndLegacyOptional(t *testing.T) {
	p, ok := ByID("google_cse")
	if !ok {
		t.Fatal("missing google_cse")
	}
	if p.Status != LegacyOptional || p.SetupGroup != "legacy_optional" {
		t.Fatalf("google_cse status/group = %s/%s", p.Status, p.SetupGroup)
	}
	if len(p.CredentialFields) != 2 {
		t.Fatalf("google_cse fields = %+v", p.CredentialFields)
	}
	reddit, ok := ByID("reddit")
	if !ok {
		t.Fatal("missing reddit")
	}
	if reddit.SetupGroup != "oauth" {
		t.Fatalf("reddit group = %s", reddit.SetupGroup)
	}
	if len(reddit.CredentialFields) < 3 {
		t.Fatalf("reddit credential fields = %+v", reddit.CredentialFields)
	}
}

func TestSortedOrdersByID(t *testing.T) {
	ps := Sorted()
	for i := 1; i < len(ps); i++ {
		if ps[i-1].ID > ps[i].ID {
			t.Fatalf("providers not sorted at %d: %s > %s", i, ps[i-1].ID, ps[i].ID)
		}
	}
}
