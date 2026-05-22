package credentials

import (
	"os"
	"testing"
)

func TestKeychainStoreGetFieldUsesEnvFallback(t *testing.T) {
	t.Setenv("FORAGE_TEST_TOKEN", "secret")
	c, err := NewKeychainStore().GetField("test", "api_key", "FORAGE_TEST_TOKEN")
	if err != nil {
		t.Fatal(err)
	}
	if !c.Found || c.Value != "secret" || c.Source != "env:FORAGE_TEST_TOKEN" {
		t.Fatalf("credential = %+v", c)
	}
}

func TestKeychainStoreGetFieldMissingWithoutEnv(t *testing.T) {
	_ = os.Unsetenv("FORAGE_TEST_MISSING")
	c, err := NewKeychainStore().GetField("missing-provider", "api_key", "FORAGE_TEST_MISSING")
	if err != nil {
		t.Fatal(err)
	}
	if c.Found {
		t.Fatalf("expected missing credential, got %+v", c)
	}
}
