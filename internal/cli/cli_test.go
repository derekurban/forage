package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/derekurban/forage/internal/apperr"
	"github.com/derekurban/forage/internal/capability"
	"github.com/derekurban/forage/internal/output"
	"github.com/spf13/cobra"
)

func runCLI(t *testing.T, args ...string) (string, error) {
	t.Helper()
	a := &app{}
	cmd := a.rootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func TestVersionDoesNotRequireConfig(t *testing.T) {
	t.Chdir(t.TempDir())
	out, err := runCLI(t, "version", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var env output.Envelope
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatal(err)
	}
	if !env.OK || env.Command != "forage version" {
		t.Fatalf("envelope = %+v", env)
	}
}

func TestCommandRequiresConfig(t *testing.T) {
	t.Chdir(t.TempDir())
	_, err := runCLI(t, "extract", "https://example.com", "--json")
	if err == nil {
		t.Fatal("expected error")
	}
	ae, ok := err.(*apperr.Error)
	if !ok {
		t.Fatalf("err = %#v", err)
	}
	if ae.Code != apperr.CodeConfigMissing {
		t.Fatalf("code = %s", ae.Code)
	}
}

func TestConfigInitAndRepairCommands(t *testing.T) {
	t.Chdir(t.TempDir())
	out, err := runCLI(t, "config", "init", "--json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(".forage", "config.yaml")); err != nil {
		t.Fatal(err)
	}
	var env output.Envelope
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatal(err)
	}
	if !env.OK {
		t.Fatalf("envelope = %+v", env)
	}
	out, err = runCLI(t, "config", "repair", "--json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatal(err)
	}
	if !env.OK {
		t.Fatalf("repair envelope = %+v", env)
	}
}

func TestProvidersListJSON(t *testing.T) {
	t.Chdir(t.TempDir())
	if _, err := runCLI(t, "config", "init"); err != nil {
		t.Fatal(err)
	}
	out, err := runCLI(t, "providers", "list", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var env output.Envelope
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatal(err)
	}
	if !env.OK || env.Data == nil {
		t.Fatalf("envelope = %+v", env)
	}
}

func TestComplementaryCommandsAreTopLevel(t *testing.T) {
	cmd := (&app{}).rootCmd()
	for _, name := range []string{"extract", "scholar", "archive"} {
		found, _, err := cmd.Find([]string{name, "--help"})
		if err != nil {
			t.Fatal(err)
		}
		if found == nil || found.Name() != name {
			t.Fatalf("missing top-level command %s", name)
		}
	}
}

func TestScholarCommandRequiresConfig(t *testing.T) {
	t.Chdir(t.TempDir())
	_, err := runCLI(t, "scholar", "openai", "--json")
	if err == nil {
		t.Fatal("expected error")
	}
	ae, ok := err.(*apperr.Error)
	if !ok || ae.Code != apperr.CodeConfigMissing {
		t.Fatalf("err = %#v", err)
	}
}

func TestExtractResponseUsesDocumentsContainer(t *testing.T) {
	data := capability.ExtractResponse{Documents: []capability.FetchResponse{{Document: capability.ExtractedDocument{URL: "https://example.com", Provider: "direct"}}}, Count: 1}
	var out bytes.Buffer
	err := (&app{opts: output.Options{JSON: true}}).writeData(commandWithOutput(&out), data, nil)
	if err != nil {
		t.Fatal(err)
	}
	var env struct {
		Data struct {
			Documents []any `json:"documents"`
			Document  any   `json:"document"`
			Count     int   `json:"count"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if len(env.Data.Documents) != 1 || env.Data.Document != nil || env.Data.Count != 1 {
		t.Fatalf("extract data shape = %s", out.String())
	}
}

func TestHumanCitationOutputIsNotRawJSON(t *testing.T) {
	resp := capability.DataResponse{Capability: capability.CitationsDOI, Data: capability.CitationResponse{
		DOI: "10.1/x", Provider: "opencitations",
		Summary: map[string]any{"coverage_note": "coverage varies"},
		Records: []capability.CitationRecord{{CitingDOI: "10.2/y", Year: 2024, Title: "Paper", Provider: "opencitations"}},
	}}
	var out bytes.Buffer
	err := (&app{}).writeData(commandWithOutput(&out), resp, nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.HasPrefix(strings.TrimSpace(out.String()), "{") || !strings.Contains(out.String(), "Citation records") {
		t.Fatalf("unexpected human output:\n%s", out.String())
	}
}

func commandWithOutput(out *bytes.Buffer) *cobra.Command {
	cmd := &cobra.Command{Use: "forage test"}
	cmd.SetOut(out)
	return cmd
}
