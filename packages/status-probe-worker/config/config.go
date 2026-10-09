package config

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"status-page/packages/core/domain"
)

type WorkerConfig struct {
	LoopSleepInterval time.Duration
	Items             []domain.StatusItemConfig
	ItemsByKey        map[string]domain.StatusItemConfig

	// Multi-pod coordination settings (EKS StatefulSet / ECS Service / K8s Deployment)
	WorkerID            string
	WorkerShardIndex    int           // Shard index (0..Total-1), auto-detected from K8s hostname ordinal if unset
	WorkerShardTotal    int           // Total worker shards (default: 1)
	WorkerLeaseEnabled  bool          // true by default
	WorkerLeaseDuration time.Duration // default 30s
}

var ordinalRegex = regexp.MustCompile(`-(\d+)$`)

// DetectHostnameOrdinal extracts trailing numeric ordinal from hostname (e.g. "worker-2" -> 2).
func DetectHostnameOrdinal(hostname string) int {
	matches := ordinalRegex.FindStringSubmatch(hostname)
	if len(matches) == 2 {
		if idx, err := strconv.Atoi(matches[1]); err == nil && idx >= 0 {
			return idx
		}
	}
	return 0
}

// LoadAndValidateConfig reads and strictly validates the status definitions and loop settings.
func LoadAndValidateConfig() (*WorkerConfig, error) {
	// 1. Loop sleep interval
	intervalStr := os.Getenv("WORKER_LOOP_INTERVAL")
	sleepInterval := 5 * time.Second
	if intervalStr != "" {
		if d, err := time.ParseDuration(intervalStr); err == nil && d > 0 {
			sleepInterval = d
		}
	}

	// 2. Multi-pod coordination settings
	shardTotal := 1
	if s := os.Getenv("WORKER_SHARD_TOTAL"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			shardTotal = n
		}
	}

	shardIndex := -1
	if s := os.Getenv("WORKER_SHARD_INDEX"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n >= 0 {
			shardIndex = n
		}
	}
	if shardIndex < 0 && shardTotal > 1 {
		// Auto-detect from Kubernetes StatefulSet hostname ordinal (e.g. status-probe-worker-2 -> 2)
		hostname, _ := os.Hostname()
		shardIndex = DetectHostnameOrdinal(hostname)
	}
	if shardIndex < 0 {
		shardIndex = 0
	}
	if shardIndex >= shardTotal {
		return nil, fmt.Errorf("WORKER_SHARD_INDEX (%d) cannot be >= WORKER_SHARD_TOTAL (%d)", shardIndex, shardTotal)
	}

	workerID := os.Getenv("WORKER_ID")
	if workerID == "" {
		hostname, err := os.Hostname()
		if err == nil && hostname != "" {
			workerID = hostname
		} else {
			workerID = fmt.Sprintf("worker-%d", time.Now().UnixNano())
		}
	}

	leaseEnabled := true
	if s := os.Getenv("WORKER_LEASE_ENABLED"); s != "" {
		s = strings.ToLower(strings.TrimSpace(s))
		if s == "false" || s == "0" || s == "no" {
			leaseEnabled = false
		}
	}

	leaseDuration := 30 * time.Second
	if s := os.Getenv("WORKER_LEASE_DURATION"); s != "" {
		if d, err := time.ParseDuration(s); err == nil && d > 0 {
			leaseDuration = d
		}
	}

	// 3. Load JSON from environment variable or file
	rawJSON := os.Getenv("STATUS_DEF_JSON")
	if rawJSON == "" {
		filePath := os.Getenv("STATUS_DEF_FILE")
		if filePath != "" {
			data, err := os.ReadFile(filePath)
			if err != nil {
				return nil, fmt.Errorf("failed to read STATUS_DEF_FILE (%s): %w", filePath, err)
			}
			rawJSON = string(data)
		}
	}

	if rawJSON == "" {
		// Provide default sample definitions if none provided in environment
		rawJSON = defaultStatusDefJSON
	}

	items, err := parseStatusItems([]byte(rawJSON))
	if err != nil {
		return nil, err
	}

	if len(items) == 0 {
		return nil, fmt.Errorf("STATUS_DEF_JSON must contain at least one status item configuration")
	}

	itemsByKey := make(map[string]domain.StatusItemConfig)
	for i, item := range items {
		if err := item.Validate(); err != nil {
			return nil, fmt.Errorf("status item [%d] validation failed: %w", i, err)
		}
		key := fmt.Sprintf("%s/%s", item.Tenant, item.Product)
		if _, exists := itemsByKey[key]; exists {
			return nil, fmt.Errorf("duplicate status item definition for tenant=%s, product=%s", item.Tenant, item.Product)
		}
		itemsByKey[key] = item
	}

	return &WorkerConfig{
		LoopSleepInterval:   sleepInterval,
		Items:               items,
		ItemsByKey:          itemsByKey,
		WorkerID:            workerID,
		WorkerShardIndex:    shardIndex,
		WorkerShardTotal:    shardTotal,
		WorkerLeaseEnabled:  leaseEnabled,
		WorkerLeaseDuration: leaseDuration,
	}, nil
}

// --- Tenant templates -------------------------------------------------------

// templateDef is a reusable product+check set, stamped out per tenant.
type templateDef struct {
	Product             string                   `json:"product"`
	PullIntervalSeconds int                      `json:"pull_interval_seconds"`
	Checks              []domain.CheckDefinition `json:"checks"`
}

// checkOverride tweaks, appends, or removes a single template check by ID.
type checkOverride struct {
	domain.CheckDefinition
	Remove bool `json:"remove,omitempty"`
}

// itemDef is one tenant binding in the document form.
type itemDef struct {
	Tenant              string                   `json:"tenant"`
	Template            string                   `json:"template,omitempty"`
	Product             string                   `json:"product,omitempty"`
	PullIntervalSeconds int                      `json:"pull_interval_seconds,omitempty"`
	Variables           map[string]string        `json:"variables,omitempty"`
	Checks              []domain.CheckDefinition `json:"checks,omitempty"`
	CheckOverrides      []checkOverride          `json:"check_overrides,omitempty"`
}

// documentDef is the object form of STATUS_DEF_JSON: shared templates + items.
type documentDef struct {
	Templates map[string]templateDef `json:"templates"`
	Items     []itemDef              `json:"items"`
}

var varPattern = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_.-]+)\s*\}\}`)

// parseStatusItems accepts either the legacy bare array of status items or the
// document form ({"templates": {...}, "items": [...]}) and returns concrete items.
func parseStatusItems(raw []byte) ([]domain.StatusItemConfig, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return nil, fmt.Errorf("STATUS_DEF_JSON is empty")
	}

	if trimmed[0] == '[' {
		var items []domain.StatusItemConfig
		if err := json.Unmarshal(raw, &items); err != nil {
			return nil, fmt.Errorf("invalid STATUS_DEF_JSON format: %w", err)
		}
		return items, nil
	}

	var doc documentDef
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("invalid STATUS_DEF_JSON document format: %w", err)
	}
	return expandDocument(&doc)
}

// expandDocument stamps out concrete status items from templates and bindings.
func expandDocument(doc *documentDef) ([]domain.StatusItemConfig, error) {
	items := make([]domain.StatusItemConfig, 0, len(doc.Items))
	for i, binding := range doc.Items {
		item, err := expandItem(binding, doc.Templates)
		if err != nil {
			return nil, fmt.Errorf("item [%d] (tenant=%q): %w", i, binding.Tenant, err)
		}
		items = append(items, *item)
	}
	return items, nil
}

func expandItem(binding itemDef, templates map[string]templateDef) (*domain.StatusItemConfig, error) {
	if binding.Tenant == "" {
		return nil, fmt.Errorf("tenant is required")
	}

	item := &domain.StatusItemConfig{
		Tenant:              binding.Tenant,
		Product:             binding.Product,
		PullIntervalSeconds: binding.PullIntervalSeconds,
	}

	var checks []domain.CheckDefinition
	if binding.Template != "" {
		tpl, ok := templates[binding.Template]
		if !ok {
			return nil, fmt.Errorf("unknown template %q", binding.Template)
		}
		if item.Product == "" {
			item.Product = tpl.Product
		}
		if item.PullIntervalSeconds == 0 {
			item.PullIntervalSeconds = tpl.PullIntervalSeconds
		}
		checks = cloneChecks(tpl.Checks)
	} else {
		checks = cloneChecks(binding.Checks)
	}

	// Variables: built-ins reflect the post-override values; item map wins on collision.
	vars := map[string]string{
		"tenant":  item.Tenant,
		"product": item.Product,
	}
	for k, v := range binding.Variables {
		vars[k] = v
	}

	if binding.Template != "" {
		merged, err := applyCheckOverrides(checks, binding.CheckOverrides)
		if err != nil {
			return nil, err
		}
		checks = merged
	} else if len(binding.CheckOverrides) > 0 {
		return nil, fmt.Errorf("check_overrides require a template")
	}

	for idx := range checks {
		if err := substituteCheckVars(&checks[idx], vars); err != nil {
			return nil, err
		}
	}
	item.Checks = checks
	return item, nil
}

// cloneChecks deep-copies check definitions so per-tenant expansions never share maps.
func cloneChecks(src []domain.CheckDefinition) []domain.CheckDefinition {
	out := make([]domain.CheckDefinition, len(src))
	for i, c := range src {
		out[i] = c
		if c.Parameters != nil {
			params := make(map[string]string, len(c.Parameters))
			for k, v := range c.Parameters {
				params[k] = v
			}
			out[i].Parameters = params
		}
	}
	return out
}

// applyCheckOverrides merges overrides by check ID: existing IDs are field-merged
// or removed, unknown IDs are appended as new checks.
func applyCheckOverrides(checks []domain.CheckDefinition, overrides []checkOverride) ([]domain.CheckDefinition, error) {
	for _, ov := range overrides {
		if ov.ID == "" {
			return nil, fmt.Errorf("check override is missing id")
		}
		pos := -1
		for i, c := range checks {
			if c.ID == ov.ID {
				pos = i
				break
			}
		}
		if ov.Remove {
			if pos == -1 {
				return nil, fmt.Errorf("cannot remove unknown check %q", ov.ID)
			}
			checks = append(checks[:pos], checks[pos+1:]...)
			continue
		}
		if pos == -1 {
			checks = append(checks, ov.CheckDefinition)
		} else {
			checks[pos] = mergeCheck(checks[pos], ov.CheckDefinition)
		}
	}
	return checks, nil
}

// mergeCheck overlays non-zero fields of ov onto base; parameters merge per-key.
func mergeCheck(base, ov domain.CheckDefinition) domain.CheckDefinition {
	if ov.Feature != "" {
		base.Feature = ov.Feature
	}
	if ov.Region != "" {
		base.Region = ov.Region
	}
	if ov.Type != "" {
		base.Type = ov.Type
	}
	if ov.Target != "" {
		base.Target = ov.Target
	}
	if ov.TimeoutMs != 0 {
		base.TimeoutMs = ov.TimeoutMs
	}
	if ov.ExpectedStatus != 0 {
		base.ExpectedStatus = ov.ExpectedStatus
	}
	if ov.ExpectedBody != "" {
		base.ExpectedBody = ov.ExpectedBody
	}
	if len(ov.Parameters) > 0 {
		if base.Parameters == nil {
			base.Parameters = make(map[string]string, len(ov.Parameters))
		}
		for k, v := range ov.Parameters {
			base.Parameters[k] = v
		}
	}
	return base
}

// substituteCheckVars replaces {{var}} placeholders in all string fields of a check.
func substituteCheckVars(c *domain.CheckDefinition, vars map[string]string) error {
	fields := []*string{&c.ID, &c.Feature, &c.Region, &c.Type, &c.Target, &c.ExpectedBody}
	for _, ptr := range fields {
		s, err := substituteVars(*ptr, vars)
		if err != nil {
			return fmt.Errorf("check %q: %w", c.ID, err)
		}
		*ptr = s
	}
	for k, v := range c.Parameters {
		s, err := substituteVars(v, vars)
		if err != nil {
			return fmt.Errorf("check %q parameter %q: %w", c.ID, k, err)
		}
		c.Parameters[k] = s
	}
	return nil
}

func substituteVars(s string, vars map[string]string) (string, error) {
	if !strings.Contains(s, "{{") {
		return s, nil
	}
	var missing string
	out := varPattern.ReplaceAllStringFunc(s, func(m string) string {
		key := varPattern.FindStringSubmatch(m)[1]
		if v, ok := vars[key]; ok {
			return v
		}
		missing = key
		return m
	})
	if missing != "" {
		return "", fmt.Errorf("unresolved variable %q", missing)
	}
	return out, nil
}

// defaultStatusDefJSON provides a built-in default configuration if none is supplied via ENV.
// It uses the document form: reusable templates stamped out for the "default" tenant.
const defaultStatusDefJSON = `{
  "templates": {
    "cloud-api": {
      "product": "cloud-api",
      "pull_interval_seconds": 10,
      "checks": [
        {
          "id": "api-gateway-check",
          "feature": "api-gateway",
          "type": "synthetic",
          "target": "https://gateway.cloud.internal/health",
          "parameters": {
            "simulated_state": "operational",
            "message": "Gateway routing active"
          }
        },
        {
          "id": "rate-limiter-check",
          "feature": "rate-limiter",
          "type": "synthetic",
          "target": "ratelimiter.cloud.internal:6379",
          "parameters": {
            "simulated_state": "operational",
            "message": "Token bucket pool healthy"
          }
        }
      ]
    },
    "auth-service": {
      "product": "auth-service",
      "pull_interval_seconds": 15,
      "checks": [
        {
          "id": "oauth-oidc-check",
          "feature": "oauth2",
          "type": "synthetic",
          "target": "https://auth.cloud.internal/.well-known/openid-configuration",
          "parameters": {
            "simulated_state": "operational",
            "message": "OIDC provider key rotation nominal"
          }
        },
        {
          "id": "session-store-check",
          "feature": "sessions",
          "type": "synthetic",
          "target": "sessions.cloud.internal:11211",
          "parameters": {
            "simulated_state": "operational",
            "message": "Cluster replication synchronised"
          }
        }
      ]
    },
    "billing-engine": {
      "product": "billing-engine",
      "pull_interval_seconds": 20,
      "checks": [
        {
          "id": "stripe-connector-check",
          "feature": "stripe-connector",
          "type": "synthetic",
          "target": "https://api.stripe.com/healthcheck",
          "parameters": {
            "simulated_state": "operational",
            "message": "Webhooks processed without delay"
          }
        },
        {
          "id": "tax-service-check-us",
          "feature": "tax-calc",
          "region": "us-east-1",
          "type": "synthetic",
          "target": "tax.us-east.cloud.internal:443",
          "parameters": {
            "simulated_state": "operational",
            "message": "US jurisdiction calculation online"
          }
        },
        {
          "id": "tax-service-check-eu",
          "feature": "tax-calc",
          "region": "eu-west-1",
          "type": "synthetic",
          "target": "tax.eu-west.cloud.internal:443",
          "parameters": {
            "simulated_state": "operational",
            "message": "EU jurisdiction calculation online"
          }
        }
      ]
    }
  },
  "items": [
    { "tenant": "default", "template": "cloud-api" },
    { "tenant": "default", "template": "auth-service" },
    { "tenant": "default", "template": "billing-engine" }
  ]
}`
