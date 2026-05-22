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
	for _, id := range []string{"brave", "jina", "browserbase", "tavily", "exa", "hackernews", "crossref", "arxiv"} {
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

func TestRegistryCredentialFields(t *testing.T) {
	brave, ok := ByID("brave")
	if !ok {
		t.Fatal("missing brave")
	}
	if len(brave.CredentialFields) != 2 {
		t.Fatalf("brave credential fields = %+v", brave.CredentialFields)
	}
	if brave.CredentialFields[0].EnvVar != "BRAVE_SEARCH_API_KEY" || !brave.CredentialFields[0].Required {
		t.Fatalf("unexpected brave search field: %+v", brave.CredentialFields[0])
	}
	if brave.CredentialFields[1].EnvVar != "BRAVE_ANSWERS_API_KEY" || brave.CredentialFields[1].Required {
		t.Fatalf("unexpected brave answers field: %+v", brave.CredentialFields[1])
	}
	orcid, ok := ByID("orcid")
	if !ok {
		t.Fatal("missing orcid")
	}
	if orcid.SetupGroup != "oauth" {
		t.Fatalf("orcid group = %s", orcid.SetupGroup)
	}
	if len(orcid.CredentialFields) != 2 {
		t.Fatalf("orcid credential fields = %+v", orcid.CredentialFields)
	}
}

func TestRemovedProvidersAreNotRegistered(t *testing.T) {
	for _, id := range []string{"blogger", "wordpress", "wordpress_com", "diffbot", "scraperapi", "google_cse", "reddit"} {
		if _, ok := ByID(id); ok {
			t.Fatalf("%s should not be registered", id)
		}
	}
}

func TestSemanticScholarUsesNoKeyPublicEndpoints(t *testing.T) {
	p, ok := ByID("semantic_scholar")
	if !ok {
		t.Fatal("missing semantic_scholar")
	}
	if p.AuthType != AuthNone {
		t.Fatalf("semantic_scholar auth = %s", p.AuthType)
	}
	if len(p.CredentialFields) != 0 {
		t.Fatalf("semantic_scholar should not require credentials: %+v", p.CredentialFields)
	}
	if p.Status != MetadataOnly {
		t.Fatalf("semantic_scholar remains metadata-only until public rate limits are reliable, got %s", p.Status)
	}
}

func TestQuotaTrackingMetadata(t *testing.T) {
	for _, id := range []string{"brave", "browserbase", "openalex", "worldnews", "serpstack"} {
		p, ok := ByID(id)
		if !ok {
			t.Fatalf("missing %s", id)
		}
		if p.Quota.Mode == "" {
			t.Fatalf("%s quota mode missing", id)
		}
		if p.Quota.Source == "" && p.ID != "direct" {
			t.Fatalf("%s quota source missing", id)
		}
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
