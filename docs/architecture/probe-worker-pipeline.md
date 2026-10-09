# Status Probe Worker Pipeline

The background worker (`packages/status-probe-worker`) is a lightweight daemon responsible for monitoring systems and evaluating health states on a periodic schedule.

---

## 🔄 Worker Lifecycle

```text
[ Startup ]
    │
    ├── 1. Read & Validate STATUS_DEF_JSON (Fail fast if invalid)
    ├── 2. Initialize StoragePort Adapter
    └── 3. Sync Unrecorded Status Items to DB (Initial zero timestamp)
    │
[ Interval Loop ] (Ticker: WORKER_LOOP_INTERVAL)
    │
    ├── 4. Search DB for Outdated Statuses
    │      WHERE last_updated < (now - pull_interval_seconds) OR last_updated IS ZERO
    │
    ├── 5. For each outdated status:
    │      ├── Concurrently execute defined checks (sync.WaitGroup)
    │      ├── Collect CheckResults (latency, success, message)
    │      ├── Aggregate results into FeatureStatus map
    │      ├── Compute overall product health (CalculateOverallState)
    │      └── Save updated Status back to StoragePort
    │
    └── 6. Sleep until next tick or handle OS signal (SIGINT, SIGTERM)
```

---

## ⚙️ 1. Startup & JSON Validation

On startup, `config.LoadAndValidateConfig()` inspects `STATUS_DEF_JSON` (or a file pointed to by `STATUS_DEF_FILE`).

Every entry must adhere to the `domain.StatusItemConfig` schema:
* `tenant`: Non-empty string
* `product`: Non-empty string
* `pull_interval_seconds`: Integer > 0
* `checks`: Non-empty array of valid check definitions

If the JSON is malformed or invalid, the daemon logs a detailed error message and **exits with non-zero status**, preventing corrupted monitoring runs.

---

## ⏱ 2. Polling Logic & Outdated Check Detection

Each configured status defines its own independent `pull_interval_seconds` (e.g. 10s for API Gateway, 60s for third-party billing).

The main worker loop wakes up on a global ticker (`WORKER_LOOP_INTERVAL`, default 5s):
1. For each status item, it queries the DB for the current record.
2. If `status.LastUpdated` is zero (newly registered item) or `time.Since(status.LastUpdated) >= pullInterval`, the status is flagged as outdated and evaluated immediately.
3. If not yet due, it is skipped without executing unnecessary checks.

---

## 🚀 3. Concurrent Check Execution

For an outdated status item containing an array of $N$ checks:
* Goroutines are spawned to execute all checks in parallel.
* Timeouts are isolated per check using `context.WithTimeout(ctx, check.TimeoutMs)`.
* Latency is measured accurately in milliseconds using monotonic timers.

---

## 🧮 4. State Aggregation Rules

Once all checks complete:
1. **Feature Aggregation:** Check results are grouped by feature identifier. If any check targeting a feature reports `outage`, the feature status escalates to `outage`. If degraded, the feature status becomes `degraded`.
2. **Product State Aggregation:** The overall product state is evaluated via `domain.CalculateOverallState(features)`:
   * **Outage:** If any feature is in `outage`.
   * **Degraded:** If any feature is `degraded` and none are in `outage`.
   * **Maintenance:** If all active features are in planned maintenance.
   * **Operational:** If operational features exist without outages or degradation.

---

## 🌐 5. Multi-Pod Coordination (EKS & ECS)

The worker supports horizontal scaling across multiple pods or tasks without duplicating check execution:

### A. AWS ECS Task Scaling & EKS Deployments (Dynamic Lease Claiming)
* When multiple workers run (e.g. an ECS Service with `desired_count: N` sharing a single Task Definition, or an EKS Deployment with `replicas: N`):
* Workers compete for outdated items using atomic lease claiming (`StoragePort.ClaimStatus`):
  * **DynamoDB:** Conditional update (`attribute_not_exists(claimed_until) OR claimed_until <= :now OR claimed_by = :worker_id`).
  * **SQLite:** Atomic conditional SQL update (`WHERE claimed_until <= :now`).
* If worker A holds the lease, workers B and C skip that item immediately. If worker A terminates mid-check, the lease automatically expires after `WORKER_LEASE_DURATION` (default 30s) and is picked up on the next cycle.

### B. EKS StatefulSets (Deterministic Hash Sharding)
* In Kubernetes StatefulSets, pods have ordinal names (`status-probe-worker-0`, `status-probe-worker-1`).
* Workers auto-detect their ordinal index from the hostname when `WORKER_SHARD_TOTAL > 1`.
* Each worker only evaluates items matching `fnv32a(tenant + "/" + product) % shard_total == shard_index`, eliminating database lock contention entirely.
