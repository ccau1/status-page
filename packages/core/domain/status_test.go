package domain_test

import (
	"testing"
	"time"

	"status-page/packages/core/domain"
)

func TestStatusItemConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  domain.StatusItemConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: domain.StatusItemConfig{
				Tenant:              "acme",
				Product:             "auth",
				PullIntervalSeconds: 30,
				Checks: []domain.CheckDefinition{
					{
						ID:      "auth-health",
						Feature: "authentication",
						Type:    "http",
						Target:  "https://auth.acme.internal/health",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing tenant",
			config: domain.StatusItemConfig{
				Tenant:              "",
				Product:             "auth",
				PullIntervalSeconds: 30,
				Checks: []domain.CheckDefinition{
					{ID: "c1", Feature: "f1", Type: "http", Target: "http://example.com"},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid interval",
			config: domain.StatusItemConfig{
				Tenant:              "acme",
				Product:             "auth",
				PullIntervalSeconds: 0,
				Checks: []domain.CheckDefinition{
					{ID: "c1", Feature: "f1", Type: "http", Target: "http://example.com"},
				},
			},
			wantErr: true,
		},
		{
			name: "missing checks",
			config: domain.StatusItemConfig{
				Tenant:              "acme",
				Product:             "auth",
				PullIntervalSeconds: 60,
				Checks:              []domain.CheckDefinition{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("expected error: %v, got: %v", tt.wantErr, err)
			}
		})
	}
}

func TestCalculateOverallState(t *testing.T) {
	now := time.Now().UTC()

	// Empty features
	if st := domain.CalculateOverallState(nil); st != domain.StateUnknown {
		t.Errorf("expected StateUnknown for empty features, got %v", st)
	}

	// All operational
	allOp := map[string]domain.FeatureStatus{
		"f1": {ID: "f1", State: domain.StateOperational, LastChecked: now},
		"f2": {ID: "f2", State: domain.StateOperational, LastChecked: now},
	}
	if st := domain.CalculateOverallState(allOp); st != domain.StateOperational {
		t.Errorf("expected StateOperational, got %v", st)
	}

	// One degraded
	degraded := map[string]domain.FeatureStatus{
		"f1": {ID: "f1", State: domain.StateOperational, LastChecked: now},
		"f2": {ID: "f2", State: domain.StateDegraded, LastChecked: now},
	}
	if st := domain.CalculateOverallState(degraded); st != domain.StateDegraded {
		t.Errorf("expected StateDegraded, got %v", st)
	}

	// Outage takes precedence over degraded
	outage := map[string]domain.FeatureStatus{
		"f1": {ID: "f1", State: domain.StateDegraded, LastChecked: now},
		"f2": {ID: "f2", State: domain.StateOutage, LastChecked: now},
	}
	if st := domain.CalculateOverallState(outage); st != domain.StateOutage {
		t.Errorf("expected StateOutage, got %v", st)
	}
}
