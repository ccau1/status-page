# `packages/core` — Shared Hexagonal Core

`packages/core` is the foundational Go module shared by both `packages/api` and `packages/status-probe-worker`. It encapsulates domain models, ports, adapters, and check registries.

---

## 📁 Directory Layout

```text
packages/core/
├── domain/                  # Pure domain entities & calculation functions
│   ├── status.go            # Status, FeatureStatus, CheckDefinition, CheckResult
│   └── status_test.go       # Domain unit tests
├── ports/                   # Inversion of control interfaces
│   └── storage.go           # StoragePort contract
├── adapters/                # Infrastructure implementations
│   ├── sqlite/              # modernc.org/sqlite adapter
│   ├── dynamodb/            # AWS SDK DynamoDB adapter
│   ├── s3/                  # AWS SDK S3 document adapter
│   └── factory/             # LoadConfigFromEnv & NewStoragePort factory
└── checks/                  # Modular health-checker system
    ├── checker.go           # Checker interface & thread-safe Registry
    ├── http.go              # HTTP/HTTPS health checker
    ├── tcp.go               # TCP socket handshake checker
    ├── dns.go               # DNS resolution checker
    └── synthetic.go         # Simulated environment checker
```

---

## 🏛 Domain Models Summary

* **`domain.StatusState`:** `operational`, `degraded`, `outage`, `maintenance`, `unknown`
* **`domain.Status`:** Complete status representation of a product under a tenant.
* **`domain.FeatureStatus`:** Granular status, latency, and messages for an individual feature.
* **`domain.CheckDefinition`:** Configured health check target and verification parameters.
* **`domain.CheckResult`:** Outcome of a check execution including timestamp and latency.
* **`domain.StatusItemConfig`:** Environment configuration schema with `Validate()` method.
* **`domain.CalculateOverallState()`:** Deterministic algorithm computing overall product health from individual feature statuses.
