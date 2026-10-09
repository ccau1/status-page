package config_test

import (
	"os"
	"testing"
	"time"

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

const templateDocJSON = `{
	"templates": {
		"standard-billing": {
			"product": "billing-engine",
			"pull_interval_seconds": 20,
			"checks": [
				{
					"id": "tax-check",
					"feature": "tax-calc",
					"region": "{{region}}",
					"type": "synthetic",
					"target": "tax.{{tenant}}.cloud.internal:443",
					"parameters": {
						"simulated_state": "operational",
						"message": "Tax online for {{tenant}}"
					}
				},
				{
					"id": "stripe-check",
					"feature": "stripe-connector",
					"type": "synthetic",
					"target": "https://api.stripe.com/healthcheck"
				}
			]
		}
	},
	"items": [
		{
			"tenant": "acme",
			"template": "standard-billing",
			"variables": { "region": "us-east-1" }
		},
		{
			"tenant": "globex",
			"template": "standard-billing",
			"pull_interval_seconds": 60,
			"variables": { "region": "eu-west-1" },
			"check_overrides": [
				{ "id": "tax-check", "timeout_ms": 8000, "parameters": { "message": "Custom EU probe" } },
				{ "id": "fraud-check", "feature": "fraud-scan", "type": "tcp", "target": "fraud.globex.internal:9000" },
				{ "id": "stripe-check", "remove": true }
			]
		}
	]
}`

func TestLoadAndValidateConfig_TemplateExpansion(t *testing.T) {
	os.Setenv("STATUS_DEF_JSON", templateDocJSON)
	defer os.Unsetenv("STATUS_DEF_JSON")

	cfg, err := config.LoadAndValidateConfig()
	if err != nil {
		t.Fatalf("expected valid document config, got error: %v", err)
	}
	if len(cfg.Items) != 2 {
		t.Fatalf("expected 2 expanded items, got %d", len(cfg.Items))
	}

	acme, ok := cfg.ItemsByKey["acme/billing-engine"]
	if !ok {
		t.Fatalf("expected acme/billing-engine in ItemsByKey, got %v", cfg.ItemsByKey)
	}
	if acme.PullIntervalSeconds != 20 {
		t.Errorf("expected acme interval 20 from template, got %d", acme.PullIntervalSeconds)
	}
	if len(acme.Checks) != 2 {
		t.Fatalf("expected 2 acme checks, got %d", len(acme.Checks))
	}
	tax := acme.Checks[0]
	if tax.Target != "tax.acme.cloud.internal:443" {
		t.Errorf("expected {{tenant}} substituted in target, got %q", tax.Target)
	}
	if tax.Region != "us-east-1" {
		t.Errorf("expected {{region}} substituted, got %q", tax.Region)
	}
	if tax.Parameters["message"] != "Tax online for acme" {
		t.Errorf("expected {{tenant}} substituted in parameters, got %q", tax.Parameters["message"])
	}

	globex := cfg.ItemsByKey["globex/billing-engine"]
	if globex.PullIntervalSeconds != 60 {
		t.Errorf("expected globex interval 60 (item override), got %d", globex.PullIntervalSeconds)
	}
	// stripe-check removed, fraud-check appended, tax-check tweaked
	if len(globex.Checks) != 2 {
		t.Fatalf("expected 2 globex checks after overrides, got %d", len(globex.Checks))
	}
	gTax := globex.Checks[0]
	if gTax.ID != "tax-check" || gTax.TimeoutMs != 8000 {
		t.Errorf("expected tax-check with timeout 8000, got id=%q timeout=%d", gTax.ID, gTax.TimeoutMs)
	}
	if gTax.Region != "eu-west-1" {
		t.Errorf("expected globex region eu-west-1, got %q", gTax.Region)
	}
	// parameters merge: overridden message wins, template key survives
	if gTax.Parameters["message"] != "Custom EU probe" {
		t.Errorf("expected merged parameter override, got %q", gTax.Parameters["message"])
	}
	if gTax.Parameters["simulated_state"] != "operational" {
		t.Errorf("expected template parameter to survive merge, got %q", gTax.Parameters["simulated_state"])
	}
	if globex.Checks[1].ID != "fraud-check" {
		t.Errorf("expected appended fraud-check, got %q", globex.Checks[1].ID)
	}
}

func TestLoadAndValidateConfig_TemplateErrors(t *testing.T) {
	cases := map[string]string{
		"unknown template": `{
			"templates": {},
			"items": [{ "tenant": "acme", "template": "nope" }]
		}`,
		"unresolved variable": `{
			"templates": { "t": { "product": "p", "pull_interval_seconds": 5, "checks": [
				{ "id": "c1", "feature": "f", "type": "tcp", "target": "{{missing}}.internal" }
			] } },
			"items": [{ "tenant": "acme", "template": "t" }]
		}`,
		"remove unknown check": `{
			"templates": { "t": { "product": "p", "pull_interval_seconds": 5, "checks": [
				{ "id": "c1", "feature": "f", "type": "tcp", "target": "x.internal" }
			] } },
			"items": [{ "tenant": "acme", "template": "t", "check_overrides": [
				{ "id": "ghost", "remove": true }
			] }]
		}`,
		"new override check incomplete": `{
			"templates": { "t": { "product": "p", "pull_interval_seconds": 5, "checks": [
				{ "id": "c1", "feature": "f", "type": "tcp", "target": "x.internal" }
			] } },
			"items": [{ "tenant": "acme", "template": "t", "check_overrides": [
				{ "id": "c2", "feature": "f2" }
			] }]
		}`,
		"all checks removed": `{
			"templates": { "t": { "product": "p", "pull_interval_seconds": 5, "checks": [
				{ "id": "c1", "feature": "f", "type": "tcp", "target": "x.internal" }
			] } },
			"items": [{ "tenant": "acme", "template": "t", "check_overrides": [
				{ "id": "c1", "remove": true }
			] }]
		}`,
		"duplicate expanded key": `{
			"templates": { "t": { "product": "p", "pull_interval_seconds": 5, "checks": [
				{ "id": "c1", "feature": "f", "type": "tcp", "target": "x.internal" }
			] } },
			"items": [
				{ "tenant": "acme", "template": "t" },
				{ "tenant": "acme", "template": "t" }
			]
		}`,
		"overrides without template": `{
			"templates": {},
			"items": [{ "tenant": "acme", "product": "p", "pull_interval_seconds": 5,
				"checks": [{ "id": "c1", "feature": "f", "type": "tcp", "target": "x.internal" }],
				"check_overrides": [{ "id": "c1", "timeout_ms": 100 }] }]
		}`,
	}

	for name, doc := range cases {
		t.Run(name, func(t *testing.T) {
			os.Setenv("STATUS_DEF_JSON", doc)
			defer os.Unsetenv("STATUS_DEF_JSON")

			if _, err := config.LoadAndValidateConfig(); err == nil {
				t.Fatalf("expected error for %s, got nil", name)
			}
		})
	}
}

func TestLoadAndValidateConfig_DefaultDocumentLoads(t *testing.T) {
	// No STATUS_DEF_JSON/FILE set: built-in default (document form) must expand.
	os.Unsetenv("STATUS_DEF_JSON")
	os.Unsetenv("STATUS_DEF_FILE")

	cfg, err := config.LoadAndValidateConfig()
	if err != nil {
		t.Fatalf("expected built-in default to load, got error: %v", err)
	}
	if len(cfg.Items) != 3 {
		t.Fatalf("expected 3 default items, got %d", len(cfg.Items))
	}
	for _, key := range []string{"default/cloud-api", "default/auth-service", "default/billing-engine"} {
		if _, ok := cfg.ItemsByKey[key]; !ok {
			t.Errorf("expected default item %s", key)
		}
	}
	// billing template carries the two regional tax checks
	billing := cfg.ItemsByKey["default/billing-engine"]
	regions := map[string]bool{}
	for _, c := range billing.Checks {
		if c.Region != "" {
			regions[c.Region] = true
		}
	}
	if !regions["us-east-1"] || !regions["eu-west-1"] {
		t.Errorf("expected regional tax checks in default billing template, got %v", regions)
	}
}

func TestLoadAndValidateConfig_InlineItemWithTenantVar(t *testing.T) {
	// Items without a template still get {{tenant}} substitution.
	doc := `{
		"templates": {},
		"items": [{
			"tenant": "acme",
			"product": "portal",
			"pull_interval_seconds": 10,
			"checks": [
				{ "id": "web-check", "feature": "web", "type": "tcp", "target": "web.{{tenant}}.internal:80" }
			]
		}]
	}`
	os.Setenv("STATUS_DEF_JSON", doc)
	defer os.Unsetenv("STATUS_DEF_JSON")

	cfg, err := config.LoadAndValidateConfig()
	if err != nil {
		t.Fatalf("expected valid inline item, got error: %v", err)
	}
	if got := cfg.ItemsByKey["acme/portal"].Checks[0].Target; got != "web.acme.internal:80" {
		t.Errorf("expected {{tenant}} substituted in inline item, got %q", got)
	}
}

func TestDetectHostnameOrdinal(t *testing.T) {
	tests := []struct {
		hostname string
		expected int
	}{
		{"status-probe-worker-0", 0},
		{"status-probe-worker-1", 1},
		{"status-probe-worker-12", 12},
		{"worker-task-3", 3},
		{"random-hostname", 0},
		{"", 0},
	}

	for _, tc := range tests {
		got := config.DetectHostnameOrdinal(tc.hostname)
		if got != tc.expected {
			t.Errorf("for hostname %q: expected %d, got %d", tc.hostname, tc.expected, got)
		}
	}
}

func TestLoadAndValidateConfig_MultiPodSettings(t *testing.T) {
	os.Setenv("WORKER_SHARD_TOTAL", "4")
	os.Setenv("WORKER_SHARD_INDEX", "2")
	os.Setenv("WORKER_ID", "custom-worker-2")
	os.Setenv("WORKER_LEASE_ENABLED", "true")
	os.Setenv("WORKER_LEASE_DURATION", "45s")
	defer func() {
		os.Unsetenv("WORKER_SHARD_TOTAL")
		os.Unsetenv("WORKER_SHARD_INDEX")
		os.Unsetenv("WORKER_ID")
		os.Unsetenv("WORKER_LEASE_ENABLED")
		os.Unsetenv("WORKER_LEASE_DURATION")
	}()

	cfg, err := config.LoadAndValidateConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.WorkerShardTotal != 4 {
		t.Errorf("expected WorkerShardTotal=4, got %d", cfg.WorkerShardTotal)
	}
	if cfg.WorkerShardIndex != 2 {
		t.Errorf("expected WorkerShardIndex=2, got %d", cfg.WorkerShardIndex)
	}
	if cfg.WorkerID != "custom-worker-2" {
		t.Errorf("expected WorkerID=custom-worker-2, got %s", cfg.WorkerID)
	}
	if !cfg.WorkerLeaseEnabled {
		t.Errorf("expected WorkerLeaseEnabled=true")
	}
	if cfg.WorkerLeaseDuration != 45*time.Second {
		t.Errorf("expected WorkerLeaseDuration=45s, got %v", cfg.WorkerLeaseDuration)
	}
}

func TestLoadAndValidateConfig_InvalidShardIndex(t *testing.T) {
	os.Setenv("WORKER_SHARD_TOTAL", "2")
	os.Setenv("WORKER_SHARD_INDEX", "2") // index cannot be >= total
	defer func() {
		os.Unsetenv("WORKER_SHARD_TOTAL")
		os.Unsetenv("WORKER_SHARD_INDEX")
	}()

	_, err := config.LoadAndValidateConfig()
	if err == nil {
		t.Fatalf("expected error when WORKER_SHARD_INDEX >= WORKER_SHARD_TOTAL")
	}
}
