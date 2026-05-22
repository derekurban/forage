package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/derekurban/forage/internal/apperr"
	"github.com/derekurban/forage/internal/output"
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
	_, err := runCLI(t, "fetch", "https://example.com", "--json")
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
