package sqlite_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"status-page/packages/core/adapters/sqlite"
	"status-page/packages/core/domain"
)

func TestSQLiteAdapter(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	adapter, err := sqlite.New(dbPath)
	if err != nil {
		t.Fatalf("failed to create sqlite adapter: %v", err)
	}
	defer adapter.Close()

	ctx := context.Background()
	if err := adapter.Init(ctx); err != nil {
		t.Fatalf("failed to init db: %v", err)
	}

	// 1. Initially empty
	st, err := adapter.GetStatus(ctx, "acme", "checkout")
	if err != nil {
		t.Fatalf("unexpected error getting non-existent status: %v", err)
	}
	if st != nil {
		t.Fatalf("expected nil for non-existent status, got: %v", st)
	}

	// 2. Save a status
	now := time.Now().UTC().Truncate(time.Second)
	testStatus := &domain.Status{
		Tenant:       "acme",
		Product:      "checkout",
		CurrentState: domain.StateOperational,
		Message:      "All systems go",
		LastUpdated:  now,
		Features: map[string]domain.FeatureStatus{
			"payments": {
				ID:          "payments",
				Name:        "Credit Card Processing",
				State:       domain.StateOperational,
				LastChecked: now,
				LatencyMs:   45,
			},
		},
		RegionalFeatures: map[string]map[string]domain.FeatureStatus{
			"eu-west-1": {
				"payments": {
					ID:          "payments",
					Name:        "Credit Card Processing",
					Region:      "eu-west-1",
					State:       domain.StateDegraded,
					LastChecked: now,
					LatencyMs:   210,
				},
			},
		},
		CheckResults: []domain.CheckResult{
			{
				CheckID:   "chk-1",
				Feature:   "payments",
				Type:      "http",
				Success:   true,
				State:     domain.StateOperational,
				LatencyMs: 45,
				Timestamp: now,
			},
		},
	}

	if err := adapter.SaveStatus(ctx, testStatus); err != nil {
		t.Fatalf("failed to save status: %v", err)
	}

	// 3. Retrieve status
	fetched, err := adapter.GetStatus(ctx, "acme", "checkout")
	if err != nil {
		t.Fatalf("failed to get status: %v", err)
	}
	if fetched == nil {
		t.Fatalf("expected status to be found")
	}
	if fetched.Tenant != "acme" || fetched.Product != "checkout" {
		t.Errorf("expected tenant acme and product checkout, got %s/%s", fetched.Tenant, fetched.Product)
	}
	if fetched.CurrentState != domain.StateOperational {
		t.Errorf("expected state operational, got %v", fetched.CurrentState)
	}
	if len(fetched.Features) != 1 || fetched.Features["payments"].LatencyMs != 45 {
		t.Errorf("features not unmarshaled correctly: %+v", fetched.Features)
	}
	euFeat, ok := fetched.RegionalFeatures["eu-west-1"]["payments"]
	if !ok || euFeat.State != domain.StateDegraded || euFeat.Region != "eu-west-1" || euFeat.LatencyMs != 210 {
		t.Errorf("regional features not unmarshaled correctly: %+v", fetched.RegionalFeatures)
	}

	// 4. List by tenant
	byTenant, err := adapter.ListStatusesByTenant(ctx, "acme")
	if err != nil || len(byTenant) != 1 {
		t.Fatalf("expected 1 status for tenant acme, got %d, err: %v", len(byTenant), err)
	}

	// 5. Test GetOutdatedStatuses
	// Outdated with olderThan 10m? Since now is fresh, it shouldn't be outdated yet
	outdated, err := adapter.GetOutdatedStatuses(ctx, 10*time.Minute)
	if err != nil {
		t.Fatalf("failed to get outdated statuses: %v", err)
	}
	if len(outdated) != 0 {
		t.Fatalf("expected 0 outdated statuses, got %d", len(outdated))
	}

	// Now update testStatus with LastUpdated 20 minutes ago
	testStatus.LastUpdated = time.Now().UTC().Add(-20 * time.Minute)
	if err := adapter.SaveStatus(ctx, testStatus); err != nil {
		t.Fatalf("failed to update status: %v", err)
	}

	outdated, err = adapter.GetOutdatedStatuses(ctx, 10*time.Minute)
	if err != nil {
		t.Fatalf("failed to get outdated statuses: %v", err)
	}
	if len(outdated) != 1 {
		t.Fatalf("expected 1 outdated status, got %d", len(outdated))
	}
}

func TestSQLiteAdapter_ClaimStatus(t *testing.T) {
	tempDB := filepath.Join(t.TempDir(), "test_claim.db")
	adapter, err := sqlite.New(tempDB)
	if err != nil {
		t.Fatalf("failed to create adapter: %v", err)
	}
	defer adapter.Close()

	ctx := context.Background()
	if err := adapter.Init(ctx); err != nil {
		t.Fatalf("failed to init adapter: %v", err)
	}

	// Insert initial status
	st := &domain.Status{
		Tenant:       "tenant-1",
		Product:      "service-a",
		CurrentState: domain.StateOperational,
		Features:     make(map[string]domain.FeatureStatus),
	}
	if err := adapter.SaveStatus(ctx, st); err != nil {
		t.Fatalf("failed to save initial status: %v", err)
	}

	// 1. Worker 1 claims status -> should succeed
	claimed, err := adapter.ClaimStatus(ctx, "tenant-1", "service-a", "worker-1", 5*time.Second)
	if err != nil {
		t.Fatalf("unexpected error claiming: %v", err)
	}
	if !claimed {
		t.Fatalf("expected worker-1 to claim successfully")
	}

	// 2. Worker 1 renews claim -> should succeed
	claimed, err = adapter.ClaimStatus(ctx, "tenant-1", "service-a", "worker-1", 5*time.Second)
	if err != nil {
		t.Fatalf("unexpected error re-claiming: %v", err)
	}
	if !claimed {
		t.Fatalf("expected worker-1 to renew successfully")
	}

	// 3. Worker 2 attempts claim while lease active -> should fail
	claimed, err = adapter.ClaimStatus(ctx, "tenant-1", "service-a", "worker-2", 5*time.Second)
	if err != nil {
		t.Fatalf("unexpected error claiming worker-2: %v", err)
	}
	if claimed {
		t.Fatalf("expected worker-2 to be rejected due to active lease")
	}

	// 4. SaveStatus clears lease -> Worker 2 can now claim
	st.LastUpdated = time.Now().UTC()
	if err := adapter.SaveStatus(ctx, st); err != nil {
		t.Fatalf("failed to save status: %v", err)
	}

	claimed, err = adapter.ClaimStatus(ctx, "tenant-1", "service-a", "worker-2", 5*time.Second)
	if err != nil {
		t.Fatalf("unexpected error claiming worker-2 after save: %v", err)
	}
	if !claimed {
		t.Fatalf("expected worker-2 to acquire claim after save reset")
	}
}
