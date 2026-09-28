# Testing & Quality Assurance

This repository maintains thorough test coverage across all layers—domain logic, storage adapters, modular checkers, HTTP handlers, worker runners, and React components.

---

## 🧪 Running Tests

### 1. Unified Test Runner
To execute the complete test suite across Go and TypeScript:
```bash
make test
```

### 2. Testing Go Packages Independently
Run tests across all Go workspace modules:
```bash
go test -v status-page/packages/core/... status-page/packages/api/... status-page/packages/status-probe-worker/...
```

Target a specific package:
```bash
# Test Hexagonal core domain, adapters, and checkers
cd packages/core && go test -v ./...

# Test API handlers
cd packages/api && go test -v ./...

# Test worker loop, config validation, and runner
cd packages/status-probe-worker && go test -v ./...
```

### 3. Testing Web Frontend (Vitest & Testing Library)
Run component, in-page filter, and routing tests:
```bash
cd packages/web
npm test
```

---

## 🔬 Test Strategy by Component

* **Domain Tests (`packages/core/domain/status_test.go`):** Validates status configuration schemas and the deterministic multi-feature state calculation algorithm (`CalculateOverallState`).
* **Check Execution Tests (`packages/core/checks/checker_test.go`):** Tests HTTP, TCP, DNS, and Synthetic checkers against test HTTP servers and custom registered checker types.
* **Storage Adapter Tests:**
  * **SQLite (`packages/core/adapters/sqlite/sqlite_test.go`):** Validates schema creation, upsert operations, tenant queries, and outdated duration filters using in-memory / temporary SQLite files.
  * **DynamoDB (`packages/core/adapters/dynamodb/dynamodb_test.go`):** Uses an in-memory mock client implementing `DynamoDBClientAPI` to verify attribute mapping, partition key queries, and scans without AWS credentials.
  * **S3 (`packages/core/adapters/s3/s3_test.go`):** Uses a mock client implementing `S3ClientAPI` to verify JSON document serialization, bucket initialization, and prefix queries.
* **Worker Runner Tests (`packages/status-probe-worker/runner/evaluator_test.go`):** Verifies the worker lifecycle, status initialization, concurrent check execution, and verifies that items within pull intervals are properly skipped on subsequent iterations.
* **Web Route & Component Tests (`packages/web/src/routes/StatusPage.test.tsx`):** Verifies hero banner status rendering, 60-day uptime tick generation, and interactive in-page feature filtering by search query and status state.
