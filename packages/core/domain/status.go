package domain

import (
	"fmt"
	"time"
)

// StatusState represents the health state of a system, product, or feature.
type StatusState string

const (
	StateOperational StatusState = "operational"
	StateDegraded    StatusState = "degraded"
	StateOutage      StatusState = "outage"
	StateMaintenance StatusState = "maintenance"
	StateUnknown     StatusState = "unknown"
)

// FeatureStatus represents the health and details of a single product feature.
type FeatureStatus struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Region      string      `json:"region,omitempty"` // set when this entry reflects a single region
	State       StatusState `json:"state"`
	LastChecked time.Time   `json:"last_checked"`
	LatencyMs   int64       `json:"latency_ms,omitempty"`
	Message     string      `json:"message,omitempty"`
}

// CheckDefinition defines a health check rule.
type CheckDefinition struct {
	ID             string            `json:"id"`
	Feature        string            `json:"feature"` // The feature key this check validates
	Region         string            `json:"region,omitempty"` // Optional region tag for per-region visibility
	Type           string            `json:"type"`    // "http", "tcp", "dns", etc.
	Target         string            `json:"target"`  // URL, hostname:port, etc.
	TimeoutMs      int               `json:"timeout_ms,omitempty"`
	ExpectedStatus int               `json:"expected_status,omitempty"` // For HTTP checks
	ExpectedBody   string            `json:"expected_body,omitempty"`
	Parameters     map[string]string `json:"parameters,omitempty"`
}

// CheckResult represents the outcome of executing a check.
type CheckResult struct {
	CheckID   string      `json:"check_id"`
	Feature   string      `json:"feature"`
	Region    string      `json:"region,omitempty"` // Region tag inherited from the check definition
	Type      string      `json:"type"`
	Success   bool        `json:"success"`
	State     StatusState `json:"state"`
	LatencyMs int64       `json:"latency_ms"`
	Message   string      `json:"message,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// Status represents the complete status view of a product for a given tenant.
type Status struct {
	Tenant       string                   `json:"tenant"`
	Product      string                   `json:"product"`
	CurrentState StatusState              `json:"current_state"`
	Message      string                   `json:"message,omitempty"`
	Features     map[string]FeatureStatus `json:"features"`
	// RegionalFeatures holds per-region feature states: region -> feature ID -> status.
	// Populated only when checks carry a region tag; Features remains the global rollup.
	RegionalFeatures map[string]map[string]FeatureStatus `json:"regional_features,omitempty"`
	LastUpdated      time.Time                           `json:"last_updated"`
	CheckResults     []CheckResult                       `json:"check_results,omitempty"`
}

// StatusItemConfig describes the configuration of a status item in the environment JSON.
type StatusItemConfig struct {
	Tenant              string            `json:"tenant"`
	Product             string            `json:"product"`
	PullIntervalSeconds int               `json:"pull_interval_seconds"`
	Checks              []CheckDefinition `json:"checks"`
}

// Validate validates that a StatusItemConfig is properly structured.
func (c *StatusItemConfig) Validate() error {
	if c.Tenant == "" {
		return fmt.Errorf("tenant is required")
	}
	if c.Product == "" {
		return fmt.Errorf("product is required")
	}
	if c.PullIntervalSeconds <= 0 {
		return fmt.Errorf("pull_interval_seconds must be greater than 0 for %s/%s", c.Tenant, c.Product)
	}
	if len(c.Checks) == 0 {
		return fmt.Errorf("at least one check must be defined for %s/%s", c.Tenant, c.Product)
	}
	for i, check := range c.Checks {
		if check.ID == "" {
			return fmt.Errorf("check[%d] id is required for %s/%s", i, c.Tenant, c.Product)
		}
		if check.Feature == "" {
			return fmt.Errorf("check[%d] feature is required for %s/%s", i, c.Tenant, c.Product)
		}
		if check.Type == "" {
			return fmt.Errorf("check[%d] type is required for %s/%s", i, c.Tenant, c.Product)
		}
		if check.Target == "" {
			return fmt.Errorf("check[%d] target is required for %s/%s", i, c.Tenant, c.Product)
		}
	}
	return nil
}

// CalculateOverallState determines the overall status state given feature statuses.
func CalculateOverallState(features map[string]FeatureStatus) StatusState {
	if len(features) == 0 {
		return StateUnknown
	}

	hasOutage := false
	hasDegraded := false
	hasMaintenance := false
	operationalCount := 0

	for _, feat := range features {
		switch feat.State {
		case StateOutage:
			hasOutage = true
		case StateDegraded:
			hasDegraded = true
		case StateMaintenance:
			hasMaintenance = true
		case StateOperational:
			operationalCount++
		}
	}

	if hasOutage {
		return StateOutage
	}
	if hasDegraded {
		return StateDegraded
	}
	if hasMaintenance && operationalCount == 0 {
		return StateMaintenance
	}
	if operationalCount > 0 {
		return StateOperational
	}
	return StateUnknown
}
