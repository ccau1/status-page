package runner_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"status-page/packages/core/adapters/sqlite"
	"status-page/packages/core/checks"
	"status-page/packages/core/domain"
	"status-page/packages/status-probe-worker/config"
	"status-page/packages/status-probe-worker/runner"
)

func TestEvaluator_ExecutionLoop(t *testing.T) {
	tempDB := filepath.Join(t.TempDir(), "worker_test.db")
	storage, err := sqlite.New(tempDB)
	if err != nil {
		t.Fatalf("failed to create sqlite: %v", err)
	}
	defer storage.Close()

	ctx := context.Background()
	if err := storage.Init(ctx); err != nil {
		t.Fatalf("failed to init storage: %v", err)
	}

	cfg := &config.WorkerConfig{
		LoopSleepInterval: 1 * time.Second,
		Items: []domain.StatusItemConfig{
			{
				Tenant:              "acme-corp",
				Product:             "identity",
				PullIntervalSeconds: 2,
				Checks: []domain.CheckDefinition{
					{
						ID:      "id-check-1",
						Feature: "login",
						Type:    "synthetic",
						Target:  "auth.acme-corp.com",
						Parameters: map[string]string{
							"simulated_state": "operational",
							"message":         "OAuth service responding normally",
						},
					},
				},
			},
		},
	}

	evaluator := runner.NewEvaluator(storage, checks.DefaultRegistry, cfg)

	// 1. SyncInitialStatuses
	if err := evaluator.SyncInitialStatuses(ctx); err != nil {
		t.Fatalf("failed to sync initial statuses: %v", err)
	}

	st, err := storage.GetStatus(ctx, "acme-corp", "identity")
	if err != nil || st == nil {
		t.Fatalf("expected initial status to exist in DB, got: %v, err: %v", st, err)
	}
	if st.CurrentState != domain.StateUnknown {
		t.Errorf("expected initial state unknown, got %v", st.CurrentState)
	}

	// 2. Run EvaluateLoopIteration (should evaluate because LastUpdated was zero)
	updated, err := evaluator.EvaluateLoopIteration(ctx)
	if err != nil {
		t.Fatalf("evaluator loop failed: %v", err)
	}
	if updated != 1 {
		t.Errorf("expected 1 item updated, got %d", updated)
	}

	// 3. Verify evaluated status in DB
	evaluated, err := storage.GetStatus(ctx, "acme-corp", "identity")
	if err != nil || evaluated == nil {
		t.Fatalf("failed to fetch evaluated status: %v", err)
	}
	if evaluated.CurrentState != domain.StateOperational {
		t.Errorf("expected evaluated state operational, got %v", evaluated.CurrentState)
	}
	if len(evaluated.Features) != 1 {
		t.Errorf("expected 1 feature, got %d", len(evaluated.Features))
	}
	loginFeat := evaluated.Features["login"]
	if loginFeat.State != domain.StateOperational {
		t.Errorf("expected login feature operational, got %v", loginFeat.State)
	}

	// 4. Immediately run again: should NOT update because pull interval is 2s and not expired yet
	updatedSecondRun, err := evaluator.EvaluateLoopIteration(ctx)
	if err != nil {
		t.Fatalf("second run failed: %v", err)
	}
	if updatedSecondRun != 0 {
		t.Errorf("expected 0 items updated because interval hasn't elapsed, got %d", updatedSecondRun)
	}
}

func TestEvaluator_RegionalAggregation(t *testing.T) {
	tempDB := filepath.Join(t.TempDir(), "worker_regional_test.db")
	storage, err := sqlite.New(tempDB)
	if err != nil {
		t.Fatalf("failed to create sqlite: %v", err)
	}
	defer storage.Close()

	ctx := context.Background()
	if err := storage.Init(ctx); err != nil {
		t.Fatalf("failed to init storage: %v", err)
	}

	cfg := &config.WorkerConfig{
		LoopSleepInterval: 1 * time.Second,
		Items: []domain.StatusItemConfig{
			{
				Tenant:              "acme-corp",
				Product:             "billing",
				PullIntervalSeconds: 2,
				Checks: []domain.CheckDefinition{
					{
						ID:      "tax-check-us",
						Feature: "tax-calc",
						Region:  "us-east-1",
						Type:    "synthetic",
						Target:  "tax.us-east.cloud.internal:443",
						Parameters: map[string]string{
							"simulated_state": "operational",
							"message":         "US region nominal",
						},
					},
					{
						ID:      "tax-check-eu",
						Feature: "tax-calc",
						Region:  "eu-west-1",
						Type:    "synthetic",
						Target:  "tax.eu-west.cloud.internal:443",
						Parameters: map[string]string{
							"simulated_state": "degraded",
							"message":         "EU upstream latency elevated",
						},
					},
					{
						ID:      "stripe-check",
						Feature: "stripe-connector",
						Type:    "synthetic",
						Target:  "https://api.stripe.com/healthcheck",
						Parameters: map[string]string{
							"simulated_state": "operational",
						},
					},
				},
			},
		},
	}

	evaluator := runner.NewEvaluator(storage, checks.DefaultRegistry, cfg)

	if err := evaluator.SyncInitialStatuses(ctx); err != nil {
		t.Fatalf("failed to sync initial statuses: %v", err)
	}
	if _, err := evaluator.EvaluateLoopIteration(ctx); err != nil {
		t.Fatalf("evaluator loop failed: %v", err)
	}

	evaluated, err := storage.GetStatus(ctx, "acme-corp", "billing")
	if err != nil || evaluated == nil {
		t.Fatalf("failed to fetch evaluated status: %v", err)
	}

	// Global rollup: worst-case across all checks, and no region tag
	globalTax := evaluated.Features["tax-calc"]
	if globalTax.State != domain.StateDegraded {
		t.Errorf("expected global tax-calc degraded (worst-case rollup), got %v", globalTax.State)
	}
	if globalTax.Region != "" {
		t.Errorf("expected global tax-calc to carry no region, got %q", globalTax.Region)
	}
	if evaluated.CurrentState != domain.StateDegraded {
		t.Errorf("expected product state degraded, got %v", evaluated.CurrentState)
	}

	// Regional breakdown
	if len(evaluated.RegionalFeatures) != 2 {
		t.Fatalf("expected 2 regions, got %d", len(evaluated.RegionalFeatures))
	}
	usFeat := evaluated.RegionalFeatures["us-east-1"]["tax-calc"]
	if usFeat.State != domain.StateOperational || usFeat.Region != "us-east-1" {
		t.Errorf("expected us-east-1 tax-calc operational with region tag, got state=%v region=%q", usFeat.State, usFeat.Region)
	}
	euFeat := evaluated.RegionalFeatures["eu-west-1"]["tax-calc"]
	if euFeat.State != domain.StateDegraded || euFeat.Region != "eu-west-1" {
		t.Errorf("expected eu-west-1 tax-calc degraded with region tag, got state=%v region=%q", euFeat.State, euFeat.Region)
	}

	// Region-less check must not appear in any regional bucket
	for region, feats := range evaluated.RegionalFeatures {
		if _, ok := feats["stripe-connector"]; ok {
			t.Errorf("region-less feature stripe-connector should not appear in region %q", region)
		}
	}

	// Check results carry the region tag
	regionByCheck := map[string]string{}
	for _, cr := range evaluated.CheckResults {
		regionByCheck[cr.CheckID] = cr.Region
	}
	if regionByCheck["tax-check-us"] != "us-east-1" || regionByCheck["tax-check-eu"] != "eu-west-1" {
		t.Errorf("expected region tags on check results, got %v", regionByCheck)
	}
	if regionByCheck["stripe-check"] != "" {
		t.Errorf("expected empty region on region-less check, got %q", regionByCheck["stripe-check"])
	}
}

func TestEvaluator_NoRegionalChecksLeavesRegionalFeaturesNil(t *testing.T) {
	tempDB := filepath.Join(t.TempDir(), "worker_no_regional_test.db")
	storage, err := sqlite.New(tempDB)
	if err != nil {
		t.Fatalf("failed to create sqlite: %v", err)
	}
	defer storage.Close()

	ctx := context.Background()
	if err := storage.Init(ctx); err != nil {
		t.Fatalf("failed to init storage: %v", err)
	}

	cfg := &config.WorkerConfig{
		LoopSleepInterval: 1 * time.Second,
		Items: []domain.StatusItemConfig{
			{
				Tenant:              "acme-corp",
				Product:             "identity",
				PullIntervalSeconds: 2,
				Checks: []domain.CheckDefinition{
					{
						ID:      "id-check-1",
						Feature: "login",
						Type:    "synthetic",
						Target:  "auth.acme-corp.com",
						Parameters: map[string]string{
							"simulated_state": "operational",
						},
					},
				},
			},
		},
	}

	evaluator := runner.NewEvaluator(storage, checks.DefaultRegistry, cfg)
	if err := evaluator.SyncInitialStatuses(ctx); err != nil {
		t.Fatalf("failed to sync initial statuses: %v", err)
	}
	if _, err := evaluator.EvaluateLoopIteration(ctx); err != nil {
		t.Fatalf("evaluator loop failed: %v", err)
	}

	evaluated, err := storage.GetStatus(ctx, "acme-corp", "identity")
	if err != nil || evaluated == nil {
		t.Fatalf("failed to fetch evaluated status: %v", err)
	}
	if evaluated.RegionalFeatures != nil {
		t.Errorf("expected RegionalFeatures to be nil without regional checks, got %v", evaluated.RegionalFeatures)
	}
}

func TestEvaluator_MultiPod_Sharding(t *testing.T) {
	tempDB := filepath.Join(t.TempDir(), "sharding_test.db")
	storage, err := sqlite.New(tempDB)
	if err != nil {
		t.Fatalf("failed to create sqlite: %v", err)
	}
	defer storage.Close()

	ctx := context.Background()
	if err := storage.Init(ctx); err != nil {
		t.Fatalf("failed to init storage: %v", err)
	}

	items := []domain.StatusItemConfig{
		{Tenant: "acme", Product: "prod-1", PullIntervalSeconds: 1, Checks: []domain.CheckDefinition{{ID: "c1", Feature: "f1", Type: "synthetic", Target: "target1"}}},
		{Tenant: "acme", Product: "prod-2", PullIntervalSeconds: 1, Checks: []domain.CheckDefinition{{ID: "c2", Feature: "f2", Type: "synthetic", Target: "target2"}}},
		{Tenant: "acme", Product: "prod-3", PullIntervalSeconds: 1, Checks: []domain.CheckDefinition{{ID: "c3", Feature: "f3", Type: "synthetic", Target: "target3"}}},
		{Tenant: "acme", Product: "prod-4", PullIntervalSeconds: 1, Checks: []domain.CheckDefinition{{ID: "c4", Feature: "f4", Type: "synthetic", Target: "target4"}}},
	}

	cfg0 := &config.WorkerConfig{
		LoopSleepInterval: 1 * time.Second,
		Items:             items,
		WorkerShardIndex:  0,
		WorkerShardTotal:  2,
	}

	cfg1 := &config.WorkerConfig{
		LoopSleepInterval: 1 * time.Second,
		Items:             items,
		WorkerShardIndex:  1,
		WorkerShardTotal:  2,
	}

	eval0 := runner.NewEvaluator(storage, checks.DefaultRegistry, cfg0)
	eval1 := runner.NewEvaluator(storage, checks.DefaultRegistry, cfg1)

	// Verify exact partitioning: every item belongs to exactly one shard
	for _, it := range items {
		r0 := eval0.IsResponsibleForShard(it.Tenant, it.Product)
		r1 := eval1.IsResponsibleForShard(it.Tenant, it.Product)
		if r0 == r1 {
			t.Fatalf("item %s/%s should belong to exactly one shard, got r0=%v, r1=%v", it.Tenant, it.Product, r0, r1)
		}
	}

	// Initial sync from each worker
	if err := eval0.SyncInitialStatuses(ctx); err != nil {
		t.Fatalf("eval0 sync failed: %v", err)
	}
	if err := eval1.SyncInitialStatuses(ctx); err != nil {
		t.Fatalf("eval1 sync failed: %v", err)
	}

	updated0, err := eval0.EvaluateLoopIteration(ctx)
	if err != nil {
		t.Fatalf("eval0 iteration failed: %v", err)
	}

	updated1, err := eval1.EvaluateLoopIteration(ctx)
	if err != nil {
		t.Fatalf("eval1 iteration failed: %v", err)
	}

	if updated0+updated1 != 4 {
		t.Fatalf("expected all 4 items updated across shards, got %d (worker 0: %d, worker 1: %d)",
			updated0+updated1, updated0, updated1)
	}
}

func TestEvaluator_MultiPod_LeasingDeduplication(t *testing.T) {
	tempDB := filepath.Join(t.TempDir(), "leasing_test.db")
	storage, err := sqlite.New(tempDB)
	if err != nil {
		t.Fatalf("failed to create sqlite: %v", err)
	}
	defer storage.Close()

	ctx := context.Background()
	if err := storage.Init(ctx); err != nil {
		t.Fatalf("failed to init storage: %v", err)
	}

	item := domain.StatusItemConfig{
		Tenant:              "acme",
		Product:             "api-gateway",
		PullIntervalSeconds: 1,
		Checks: []domain.CheckDefinition{
			{ID: "c1", Feature: "edge", Type: "synthetic", Target: "gateway.internal"},
		},
	}

	cfgA := &config.WorkerConfig{
		LoopSleepInterval:   1 * time.Second,
		Items:               []domain.StatusItemConfig{item},
		WorkerID:            "task-A",
		WorkerLeaseEnabled:  true,
		WorkerLeaseDuration: 10 * time.Second,
	}

	cfgB := &config.WorkerConfig{
		LoopSleepInterval:   1 * time.Second,
		Items:               []domain.StatusItemConfig{item},
		WorkerID:            "task-B",
		WorkerLeaseEnabled:  true,
		WorkerLeaseDuration: 10 * time.Second,
	}

	evalA := runner.NewEvaluator(storage, checks.DefaultRegistry, cfgA)
	evalB := runner.NewEvaluator(storage, checks.DefaultRegistry, cfgB)

	// Sync initial status
	if err := evalA.SyncInitialStatuses(ctx); err != nil {
		t.Fatalf("failed initial sync: %v", err)
	}

	// Task A acquires lease in storage
	claimed, err := storage.ClaimStatus(ctx, item.Tenant, item.Product, "task-A", 10*time.Second)
	if err != nil || !claimed {
		t.Fatalf("task-A failed to claim lease: %v, claimed=%v", err, claimed)
	}

	// Task B attempts to run evaluation loop while Task A holds the lease
	updatedByB, err := evalB.EvaluateLoopIteration(ctx)
	if err != nil {
		t.Fatalf("evalB failed: %v", err)
	}
	if updatedByB != 0 {
		t.Fatalf("expected Task B to skip evaluation due to active lease held by Task A, but updated %d", updatedByB)
	}

	// Task A executes evaluation and completes
	updatedByA, err := evalA.EvaluateLoopIteration(ctx)
	if err != nil {
		t.Fatalf("evalA failed: %v", err)
	}
	if updatedByA != 1 {
		t.Fatalf("expected Task A to evaluate its owned item, got %d", updatedByA)
	}
}
