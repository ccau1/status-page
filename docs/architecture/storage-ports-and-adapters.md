# Hexagonal Storage Ports & Adapters

The storage subsystem abstracts data access behind a single Go interface: `ports.StoragePort`. Three production-ready adapters are provided out of the box, selected via the `STORAGE_ENGINE` environment variable.

---

## 🔌 The `StoragePort` Interface

Located in `packages/core/ports/storage.go`:

```go
type StoragePort interface {
    // Init initializes schema tables, buckets, or collections if they don't exist.
    Init(ctx context.Context) error

    // GetStatus retrieves status for a specific tenant and product.
    GetStatus(ctx context.Context, tenant string, product string) (*domain.Status, error)

    // ListStatusesByTenant retrieves all product statuses for a single tenant.
    ListStatusesByTenant(ctx context.Context, tenant string) ([]domain.Status, error)

    // ListAllStatuses retrieves all product statuses across all tenants.
    ListAllStatuses(ctx context.Context) ([]domain.Status, error)

    // GetOutdatedStatuses retrieves records whose LastUpdated is older than the given duration.
    GetOutdatedStatuses(ctx context.Context, olderThan time.Duration) ([]domain.Status, error)

    // SaveStatus persists or updates a status record.
    SaveStatus(ctx context.Context, status *domain.Status) error

    // Close releases database handles and network connections.
    Close() error
}
```

---

## 🏭 Storage Factory Pattern

Located in `packages/core/adapters/factory/factory.go`:

```go
cfg := factory.LoadConfigFromEnv()
storage, err := factory.NewStoragePort(ctx, cfg)
```

The factory reads standard environment variables:
* `STORAGE_ENGINE`: `sqlite` (default), `dynamodb`, or `s3`
* Calls `adapter.Init(ctx)` automatically to verify connectivity and create tables/buckets before returning.

---

## 📦 Adapters

### 1. SQLite Adapter (`packages/core/adapters/sqlite/`)
* **Driver:** `modernc.org/sqlite` (100% pure Go, no CGO or gcc required).
* **Target:** Local development, Docker Compose environments, embedded appliances.
* **Schema:**
  ```sql
  CREATE TABLE IF NOT EXISTS statuses (
      tenant TEXT NOT NULL,
      product TEXT NOT NULL,
      current_state TEXT NOT NULL,
      message TEXT,
      features_json TEXT NOT NULL,
      check_results_json TEXT,
      last_updated INTEGER NOT NULL,
      PRIMARY KEY(tenant, product)
  );
  CREATE INDEX IF NOT EXISTS idx_statuses_last_updated ON statuses(last_updated);
  ```
* **Concurrency:** Configured with `db.SetMaxOpenConns(1)` to ensure write safety under high concurrent loop execution.

### 2. AWS DynamoDB Adapter (`packages/core/adapters/dynamodb/`)
* **SDK:** AWS SDK for Go v2 (`github.com/aws/aws-sdk-go-v2/service/dynamodb`).
* **Target:** Cloud-native deployments with high availability and serverless scaling.
* **Key Schema:**
  * Partition Key (`PK`): `TENANT#{tenant}` (e.g. `TENANT#default`)
  * Sort Key (`SK`): `PRODUCT#{product}` (e.g. `PRODUCT#cloud-api`)
* **Attributes:**
  * `current_state`: string
  * `features_json`: JSON string of feature states
  * `check_results_json`: JSON string of recent checks
  * `last_updated`: Unix epoch timestamp integer
* **Auto-Provisioning:** If the table does not exist, `Init(ctx)` creates it automatically with `PAY_PER_REQUEST` billing mode.

### 3. AWS S3 Adapter (`packages/core/adapters/s3/`)
* **SDK:** AWS SDK for Go v2 (`github.com/aws/aws-sdk-go-v2/service/s3`).
* **Target:** Low-cost, read-heavy public status pages served via CloudFront or CDN edge caching.
* **Object Key Convention:** `statuses/{tenant}/{product}.json`
* **Format:** Formatted JSON document of `domain.Status`.
* **Auto-Provisioning:** `Init(ctx)` validates the bucket via `HeadBucket` and creates it if absent.
