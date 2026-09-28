package config_test

import (
	"os"
	"testing"

	"status-page/packages/status-probe-worker/config"
)

func TestLoadAndValidateConfig_Valid(t *testing.T) {
	validJSON := `[
		{
			"tenant": "test-tenant",
			"product": "test-product",
			"pull_interval_seconds": 10,
			"checks": [
				{
					"id": "chk-1",
					"feature": "search",
					"type": "synthetic",
					"target": "search.internal"
				}
			]
		}
	]`
	os.Setenv("STATUS_DEF_JSON", validJSON)
	defer os.Unsetenv("STATUS_DEF_JSON")

	cfg, err := config.LoadAndValidateConfig()
	if err != nil {
		t.Fatalf("expected valid config, got error: %v", err)
	}
	if len(cfg.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(cfg.Items))
	}
	if cfg.Items[0].Tenant != "test-tenant" {
		t.Errorf("expected tenant test-tenant, got %s", cfg.Items[0].Tenant)
	}
}

func TestLoadAndValidateConfig_InvalidJSON(t *testing.T) {
	os.Setenv("STATUS_DEF_JSON", `{"invalid": json}`)
	defer os.Unsetenv("STATUS_DEF_JSON")

	_, err := config.LoadAndValidateConfig()
	if err == nil {
		t.Fatalf("expected error on invalid JSON, got nil")
	}
}

func TestLoadAndValidateConfig_MissingFields(t *testing.T) {
	os.Setenv("STATUS_DEF_JSON", `[{"tenant": "no-product", "pull_interval_seconds": 10}]`)
	defer os.Unsetenv("STATUS_DEF_JSON")

	_, err := config.LoadAndValidateConfig()
	if err == nil {
		t.Fatalf("expected error on missing product/checks, got nil")
	}
}
