package runner

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"status-page/packages/core/checks"
	"status-page/packages/core/domain"
	"status-page/packages/core/ports"
	"status-page/packages/status-probe-worker/config"
)

type Evaluator struct {
	storage  ports.StoragePort
	registry *checks.Registry
	cfg      *config.WorkerConfig
}

func NewEvaluator(storage ports.StoragePort, reg *checks.Registry, cfg *config.WorkerConfig) *Evaluator {
	if reg == nil {
		reg = checks.DefaultRegistry
	}
	return &Evaluator{
		storage:  storage,
		registry: reg,
		cfg:      cfg,
	}
}

// SyncInitialStatuses ensures all configured status items exist in the database.
func (e *Evaluator) SyncInitialStatuses(ctx context.Context) error {
	for _, item := range e.cfg.Items {
		existing, err := e.storage.GetStatus(ctx, item.Tenant, item.Product)
		if err != nil {
			return fmt.Errorf("failed to check existing status for %s/%s: %w", item.Tenant, item.Product, err)
		}
		if existing == nil {
			log.Printf("[Worker] Initializing unrecorded status item for tenant=%s product=%s", item.Tenant, item.Product)
			initStatus := &domain.Status{
				Tenant:       item.Tenant,
				Product:      item.Product,
				CurrentState: domain.StateUnknown,
				Message:      "Initializing status evaluation...",
				Features:     make(map[string]domain.FeatureStatus),
				LastUpdated:  time.Time{}, // Zero time to ensure immediate evaluation on first tick
			}
			if err := e.storage.SaveStatus(ctx, initStatus); err != nil {
				return fmt.Errorf("failed to save initial status for %s/%s: %w", item.Tenant, item.Product, err)
			}
		}
	}
	return nil
}

// EvaluateLoopIteration performs one cycle of searching outdated statuses in DB and updating them.
func (e *Evaluator) EvaluateLoopIteration(ctx context.Context) (int, error) {
	updatedCount := 0

	for _, item := range e.cfg.Items {
		st, err := e.storage.GetStatus(ctx, item.Tenant, item.Product)
		if err != nil {
			log.Printf("[Worker] Error getting status for %s/%s: %v", item.Tenant, item.Product, err)
			continue
		}

		pullInterval := time.Duration(item.PullIntervalSeconds) * time.Second
		isOutdated := st == nil || st.LastUpdated.IsZero() || time.Since(st.LastUpdated) >= pullInterval

		if !isOutdated {
			continue
		}

		if err := e.evaluateSingleProduct(ctx, item); err != nil {
			log.Printf("[Worker] Failed to evaluate %s/%s: %v", item.Tenant, item.Product, err)
			continue
		}

		updatedCount++
	}

	return updatedCount, nil
}

func (e *Evaluator) evaluateSingleProduct(ctx context.Context, item domain.StatusItemConfig) error {
	results := make([]domain.CheckResult, len(item.Checks))
	var wg sync.WaitGroup

	for i, chkDef := range item.Checks {
		wg.Add(1)
		go func(idx int, def domain.CheckDefinition) {
			defer wg.Done()
			results[idx] = checks.ExecuteCheck(ctx, e.registry, def)
		}(i, chkDef)
	}

	wg.Wait()

	// Aggregate check results into features
	features := make(map[string]domain.FeatureStatus)
	for _, res := range results {
		feat, exists := features[res.Feature]
		if !exists {
			feat = domain.FeatureStatus{
				ID:          res.Feature,
				Name:        formatFeatureName(res.Feature),
				State:       res.State,
				LastChecked: res.Timestamp,
				LatencyMs:   res.LatencyMs,
				Message:     res.Message,
			}
		} else {
			// If existing is operational but this check failed, escalate state
			if res.State == domain.StateOutage {
				feat.State = domain.StateOutage
				feat.Message = res.Message
			} else if res.State == domain.StateDegraded && feat.State != domain.StateOutage {
				feat.State = domain.StateDegraded
				feat.Message = res.Message
			}
			if res.LatencyMs > feat.LatencyMs {
				feat.LatencyMs = res.LatencyMs
			}
			feat.LastChecked = res.Timestamp
		}
		features[res.Feature] = feat
	}

	overallState := domain.CalculateOverallState(features)
	overallMsg := "All systems operational"
	switch overallState {
	case domain.StateOutage:
		overallMsg = "Major outage detected on core capabilities"
	case domain.StateDegraded:
		overallMsg = "Experiencing degraded performance on select features"
	case domain.StateMaintenance:
		overallMsg = "Under scheduled maintenance"
	}

	newStatus := &domain.Status{
		Tenant:       item.Tenant,
		Product:      item.Product,
		CurrentState: overallState,
		Message:      overallMsg,
		Features:     features,
		CheckResults: results,
		LastUpdated:  time.Now().UTC(),
	}

	if err := e.storage.SaveStatus(ctx, newStatus); err != nil {
		return fmt.Errorf("failed to save evaluated status: %w", err)
	}

	log.Printf("[Worker] Evaluated %s/%s -> State: %s (%d features, %d checks)",
		item.Tenant, item.Product, overallState, len(features), len(results))

	return nil
}

func formatFeatureName(raw string) string {
	if len(raw) == 0 {
		return ""
	}
	// Convert hyphens to spaces and title case simple words
	out := []rune(raw)
	capitalize := true
	for i, r := range out {
		if r == '-' || r == '_' {
			out[i] = ' '
			capitalize = true
			continue
		}
		if capitalize && r >= 'a' && r <= 'z' {
			out[i] = r - ('a' - 'A')
			capitalize = false
		} else {
			capitalize = false
		}
	}
	return string(out)
}
