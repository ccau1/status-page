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
