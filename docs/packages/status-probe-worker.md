# `packages/status-probe-worker` — Status Probing Daemon

The `status-probe-worker` package is a standalone Go daemon that continually evaluates health checks and keeps status records up to date.

---

## 🛠 Configuration Schema

Status items are passed via the `STATUS_DEF_JSON` environment variable (or referenced by filepath with `STATUS_DEF_FILE`).

### Example JSON Specification:
```json
[
  {
    "tenant": "default",
    "product": "cloud-api",
    "pull_interval_seconds": 10,
    "checks": [
      {
        "id": "gateway-check",
        "feature": "api-gateway",
        "type": "http",
        "target": "https://gateway.example.com/health",
        "timeout_ms": 3000,
        "expected_status": 200
      },
      {
        "id": "ratelimit-check",
        "feature": "rate-limiter",
        "type": "tcp",
        "target": "redis.internal:6379",
        "timeout_ms": 1000
      }
    ]
  }
]
```

---

## 🔁 Runner Engine (`runner/evaluator.go`)

### Initialization: `SyncInitialStatuses`
Ensures that all configured status items exist in the database upon startup. If a status record does not yet exist, an uninitialized entry with a zero timestamp is created, triggering immediate evaluation on the very first cycle.

### Loop Execution: `EvaluateLoopIteration`
1. Iterates over all configured products.
2. Checks whether `time.Since(status.LastUpdated) >= pullInterval` or `status.LastUpdated.IsZero()`.
3. If outdated:
   * Concurrently runs all checks defined for the product using goroutines.
   * Matches checks to their corresponding features.
   * Computes individual feature health and overall product state.
   * Persists the updated snapshot via the configured `StoragePort`.
4. Returns the number of status items updated in the iteration.
