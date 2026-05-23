package forage

import (
	"testing"

	"github.com/derekurban/forage/internal/capability"
)

func TestPublicAliasesCompile(t *testing.T) {
	req := SearchRequest{Query: "x", Capability: capability.SearchWeb}
	if req.Query != "x" {
		t.Fatalf("request = %+v", req)
	}
	var _ DataRequest = DataRequest{Capability: capability.ArchiveLookup}
	var _ EvidencePack
	var _ GatherRequest = GatherRequest{Query: "x"}
	var _ RetrieveRequest = RetrieveRequest{Input: "https://example.com"}
	var _ BriefRequest = BriefRequest{Query: "x"}
}
