package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitCreatesGlobalConfigWithoutSecrets(t *testing.T) {
	t.Chdir(t.TempDir())
	path, created, err := Init(false)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if !created {
		t.Fatal("Init() did not create config")
	}
	if path != filepath.Join(".forage", "config.yaml") {
		t.Fatalf("path = %q", path)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) == "" {
		t.Fatal("config is empty")
	}
	if Exists() != true {
		t.Fatal("Exists() = false")
	}
}

func TestLoadAppliesDefaults(t *testing.T) {
	t.Chdir(t.TempDir())
	if _, _, err := Init(false); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Profile != "free" {
		t.Fatalf("Profile = %q", cfg.Profile)
	}
	if cfg.Credentials.Store != "keychain" {
		t.Fatalf("Credentials.Store = %q", cfg.Credentials.Store)
	}
	if !Enabled(cfg, "brave") {
		t.Fatal("brave should be enabled by default")
	}
	if !Enabled(cfg, "direct") {
		t.Fatal("direct should be enabled by default")
	}
	if Enabled(cfg, "google_cse") {
		t.Fatal("google_cse should be legacy optional and disabled by default")
	}
}

func TestLoadMergesNewDefaultProviders(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.MkdirAll(".forage", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(".forage", "config.yaml"), []byte("version: 1\nproviders:\n  brave:\n    enabled: true\nrouting: {}\ncache:\n  database: .forage/state.db\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !Enabled(cfg, "direct") {
		t.Fatal("direct should be merged into older configs")
	}
	if len(cfg.Routing["fetch.url"]) == 0 {
		t.Fatal("default fetch route should be merged")
	}
}
