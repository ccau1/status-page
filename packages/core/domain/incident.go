package domain

import "time"

// IncidentSeverity denotes the impact level of an incident.
type IncidentSeverity string

const (
	SeverityCritical    IncidentSeverity = "critical"
	SeverityMajor       IncidentSeverity = "major"
	SeverityMinor       IncidentSeverity = "minor"
	SeverityMaintenance IncidentSeverity = "maintenance"
	SeverityInfo        IncidentSeverity = "info"
)

// IncidentState denotes the lifecycle stage of an incident investigation.
type IncidentState string

const (
	IncidentInvestigating IncidentState = "investigating"
	IncidentIdentified    IncidentState = "identified"
	IncidentMonitoring    IncidentState = "monitoring"
	IncidentResolved      IncidentState = "resolved"
)

// IncidentUpdate represents an update entry in an ongoing incident timeline.
type IncidentUpdate struct {
	Timestamp time.Time     `json:"timestamp"`
	State     IncidentState `json:"state"`
	Message   string        `json:"message"`
}

// Incident represents an operator announcement or tracked service disruption banner.
// Incidents can link to one or more products and/or specific health checks/features.
type Incident struct {
	ID        string           `json:"id"`
	Tenant    string           `json:"tenant"`             // e.g. "default", "acme", or "*" for global
	Product   string           `json:"product,omitempty"`  // backward-compatible single product slug
	Products  []string         `json:"products,omitempty"` // linked products (e.g. ["cloud-api", "billing-engine"])
	CheckIDs  []string         `json:"check_ids,omitempty"`// linked check IDs (e.g. ["gateway-check", "tax-service-check"])
	Features  []string         `json:"features,omitempty"` // linked feature IDs (e.g. ["api-gateway", "tax-calc"])
	Title     string           `json:"title"`
	Severity  IncidentSeverity `json:"severity"`
	State     IncidentState    `json:"state"`
	Message   string           `json:"message"`
	Active    bool             `json:"active"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
	Updates   []IncidentUpdate `json:"updates,omitempty"`
}

// AppliesToProduct checks if an incident applies to a specific product.
func (inc *Incident) AppliesToProduct(productSlug string) bool {
	if productSlug == "" {
		return true
	}
	if len(inc.Products) == 0 && inc.Product == "" {
		return true // Global platform-wide incident applies to all
	}
	if inc.Product == productSlug {
		return true
	}
	for _, p := range inc.Products {
		if p == productSlug {
			return true
		}
	}
	return false
}

// AppliesToCheck checks if an incident explicitly links to a given check ID or feature ID.
func (inc *Incident) AppliesToCheck(checkID, featureID string) bool {
	for _, c := range inc.CheckIDs {
		if c == checkID {
			return true
		}
	}
	for _, f := range inc.Features {
		if f == featureID {
			return true
		}
	}
	return false
}
