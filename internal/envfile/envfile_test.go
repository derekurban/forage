package envfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFileLoadsRepoEnv(t *testing.T) {
	t.Chdir(t.TempDir())
	path := filepath.Join(".", ".env")
	if err := os.WriteFile(path, []byte("FORAGE_TEST_ENV=loaded\nQUOTED=\"hello world\"\nCOMMENTED=value # local note\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FORAGE_TEST_ENV", "")
	_ = os.Unsetenv("FORAGE_TEST_ENV")
	_ = os.Unsetenv("QUOTED")
	_ = os.Unsetenv("COMMENTED")
	if err := LoadFile(path); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("FORAGE_TEST_ENV"); got != "loaded" {
		t.Fatalf("FORAGE_TEST_ENV = %q", got)
	}
	if got := os.Getenv("QUOTED"); got != "hello world" {
		t.Fatalf("QUOTED = %q", got)
	}
	if got := os.Getenv("COMMENTED"); got != "value" {
		t.Fatalf("COMMENTED = %q", got)
	}
}

func TestLoadFileDoesNotOverrideExistingEnv(t *testing.T) {
	t.Chdir(t.TempDir())
	path := filepath.Join(".", ".env")
	if err := os.WriteFile(path, []byte("FORAGE_EXISTING=from_file\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FORAGE_EXISTING", "from_process")
	if err := LoadFile(path); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("FORAGE_EXISTING"); got != "from_process" {
		t.Fatalf("FORAGE_EXISTING = %q", got)
	}
}

func TestLoadFileIgnoresMissingEnv(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := LoadFile(".env"); err != nil {
		t.Fatal(err)
	}
}
