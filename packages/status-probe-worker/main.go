package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"status-page/packages/core/adapters/factory"
	"status-page/packages/core/checks"
	"status-page/packages/core/envutil"
	"status-page/packages/status-probe-worker/config"
	"status-page/packages/status-probe-worker/runner"
)

func main() {
	envutil.LoadDotEnv()

	log.Println("[Worker] Starting status-probe-worker...")

	// 1. Validate environment JSON configuration on startup
	cfg, err := config.LoadAndValidateConfig()
	if err != nil {
		log.Fatalf("[Worker] Configuration error: invalid status item definitions: %v", err)
	}
	log.Printf("[Worker] Validated configuration successfully: %d status items defined, loop interval: %v",
		len(cfg.Items), cfg.LoopSleepInterval)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 2. Initialize storage adapter
	storageCfg := factory.LoadConfigFromEnv()
	log.Printf("[Worker] Connecting to storage engine: %s", storageCfg.Engine)
	storage, err := factory.NewStoragePort(ctx, storageCfg)
	if err != nil {
		log.Fatalf("[Worker] Failed to connect to storage: %v", err)
	}
	defer storage.Close()

	// 3. Initialize Evaluator and sync status items
	evaluator := runner.NewEvaluator(storage, checks.DefaultRegistry, cfg)
	if err := evaluator.SyncInitialStatuses(ctx); err != nil {
		log.Fatalf("[Worker] Failed to sync initial statuses into DB: %v", err)
	}

	// 4. Setup graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	log.Printf("[Worker] Starting interval evaluation loop (interval: %v)...", cfg.LoopSleepInterval)
	ticker := time.NewTicker(cfg.LoopSleepInterval)
	defer ticker.Stop()

	// Run initial evaluation immediately
	if updated, err := evaluator.EvaluateLoopIteration(ctx); err != nil {
		log.Printf("[Worker] Initial evaluation cycle error: %v", err)
	} else {
		log.Printf("[Worker] Initial cycle completed: %d items updated", updated)
	}

	// Main evaluation loop
	for {
		select {
		case <-stop:
			log.Println("[Worker] Shutdown signal received. Exiting gracefully...")
			return
		case <-ticker.C:
			updated, err := evaluator.EvaluateLoopIteration(ctx)
			if err != nil {
				log.Printf("[Worker] Evaluation cycle encountered error: %v", err)
			} else if updated > 0 {
				log.Printf("[Worker] Loop cycle updated %d status items", updated)
			}
		}
	}
}
