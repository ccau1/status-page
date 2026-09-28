package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"status-page/packages/core/domain"
)

type WorkerConfig struct {
	LoopSleepInterval time.Duration
	Items             []domain.StatusItemConfig
	ItemsByKey        map[string]domain.StatusItemConfig
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

	// 2. Load JSON from environment variable or file
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

	var items []domain.StatusItemConfig
	if err := json.Unmarshal([]byte(rawJSON), &items); err != nil {
		return nil, fmt.Errorf("invalid STATUS_DEF_JSON format: %w", err)
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
		LoopSleepInterval: sleepInterval,
		Items:             items,
		ItemsByKey:        itemsByKey,
	}, nil
}

// defaultStatusDefJSON provides a built-in default configuration if none is supplied via ENV
const defaultStatusDefJSON = `[
  {
    "tenant": "default",
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
  {
    "tenant": "default",
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
  {
    "tenant": "default",
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
        "id": "tax-service-check",
        "feature": "tax-calc",
        "type": "synthetic",
        "target": "tax.cloud.internal:443",
        "parameters": {
          "simulated_state": "operational",
          "message": "Jurisdiction calculation online"
        }
      }
    ]
  }
]`
