package providers

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

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

func TestEnvExampleCoversCredentialFields(t *testing.T) {
	b, err := os.ReadFile("../../.env.example")
	if err != nil {
		t.Fatal(err)
	}
	envNames := map[string]bool{}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, _, ok := strings.Cut(line, "=")
		if ok {
			envNames[strings.TrimSpace(name)] = true
		}
	}
	for _, p := range Registry() {
		for _, f := range p.CredentialFields {
			if f.EnvVar == "" {
				continue
			}
			if !envNames[f.EnvVar] {
				t.Fatalf("%s.%s env var %s missing from .env.example", p.ID, f.Name, f.EnvVar)
			}
		}
	}
}

func TestEnvExampleContainsNoRealLookingSecrets(t *testing.T) {
	b, err := os.ReadFile("../../.env.example")
	if err != nil {
		t.Fatal(err)
	}
	re := regexp.MustCompile(`(?m)^[A-Z0-9_]+=(.+)$`)
	for _, m := range re.FindAllStringSubmatch(string(b), -1) {
		value := strings.TrimSpace(m[1])
		if value == "" {
			continue
		}
		if strings.Contains(value, "YOUR_REDDIT_USERNAME") {
			continue
		}
		t.Fatalf(".env.example contains non-placeholder value %q", value)
	}
}

func TestUnpaywallEmailCredential(t *testing.T) {
	p, ok := ByID("unpaywall")
	if !ok {
		t.Fatal("missing unpaywall")
	}
	if len(p.CredentialFields) != 1 {
		t.Fatalf("unpaywall fields = %+v", p.CredentialFields)
	}
	f := p.CredentialFields[0]
	if f.EnvVar != "UNPAYWALL_EMAIL" || f.Secret || !f.Required {
		t.Fatalf("unexpected unpaywall field: %+v", f)
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
