package envutil_test

import (
	"os"
	"path/filepath"
	"testing"

	"status-page/packages/core/envutil"
)

func TestLoadFile(t *testing.T) {
	tempDir := t.TempDir()
	envPath := filepath.Join(tempDir, ".env")

	content := `
# Sample comment
TEST_SIMPLE_VAR=hello_world
TEST_QUOTED_VAR="quoted value"

TEST_JSON='[
  {
    "tenant": "test-env-tenant",
    "product": "test-product"
  }
]'
`
	if err := os.WriteFile(envPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test .env: %v", err)
	}

	// Ensure clean state
	os.Unsetenv("TEST_SIMPLE_VAR")
	os.Unsetenv("TEST_QUOTED_VAR")
	os.Unsetenv("TEST_JSON")

	if err := envutil.LoadFile(envPath); err != nil {
		t.Fatalf("LoadFile failed: %v", err)
	}

	if os.Getenv("TEST_SIMPLE_VAR") != "hello_world" {
		t.Errorf("expected hello_world, got %s", os.Getenv("TEST_SIMPLE_VAR"))
	}
	if os.Getenv("TEST_QUOTED_VAR") != "quoted value" {
		t.Errorf("expected 'quoted value', got %s", os.Getenv("TEST_QUOTED_VAR"))
	}
	if len(os.Getenv("TEST_JSON")) == 0 {
		t.Errorf("expected non-empty multiline TEST_JSON")
	}
}
