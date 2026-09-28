# Platform Status Page

A distributed, multi-tenant platform status and monitoring system.

## 📖 Detailed Documentation

Comprehensive documentation with nested guides is available in the **[`/docs`](./docs/)** directory:
* [Architecture Overview](./docs/architecture/overview.md) & [Hexagonal Storage Ports](./docs/architecture/storage-ports-and-adapters.md)
* [Worker Pipeline](./docs/architecture/probe-worker-pipeline.md) & [Modular Checks System](./docs/architecture/modular-checks.md)
* [Package Details](./docs/packages/) (`web`, `api`, `core`, `status-probe-worker`)
* [Getting Started Guide](./docs/guides/getting-started.md) & [Docker Hot Reload Guide](./docs/guides/docker-and-hot-reload.md)

---

## 🏗 Architecture & Packages

```text
status-page/
├── docs/                     # Comprehensive nested documentation
├── packages/
│   ├── web/                  # Vite SSR React frontend & /admin incident console
│   ├── api/                  # Go REST API gateway (Hexagonal Architecture)
│   ├── auth/                 # Standalone stateless gRPC authentication microservice
│   ├── core/                 # Shared Go module (Domain, Ports, & Adapters)
│   └── status-probe-worker/   # Interval background worker daemon
├── docker-compose.yml        # Orchestration with live hot reloading (4 containers)
├── Makefile                  # Developer workflow targets (make up, make down)
└── go.work                   # Go multi-module workspace
```

### 1. `packages/web` (Vite SSR + React)
- **Framework:** React with TypeScript, Vite SSR, and React Router.
- **Public Routing:** Supports `/{tenant?}/{locale?}/status/{product?}` with instant in-page feature filtering.
- **Internal Admin Portal:** `/admin` Incident Control Center with dual-mode authentication (Enterprise SSO and Username/Password), timeline progress logs, and banner controls.
- **Styling:** Custom Obsidian Glassmorphism theme using Vanilla CSS (no Tailwind dependency), animated pulse indicators, latency badges, and 60-day procedural uptime tracks.

### 2. `packages/auth` (Stateless gRPC Microservice)
- **Stateless Operation:** Issues and verifies HMAC-SHA256 signed JWTs with zero database dependencies.
- **Multi-Provider OIDC:** Connects to Microsoft Entra ID (Azure AD), Okta, and offline mock dev SSO.
- **Credentials Auth:** Supports local administrator username & password verification (`crypto/subtle.ConstantTimeCompare`).
- **Protobuf / gRPC Engine:** Exposes `Login`, `Register`, `Logout`, `ForgotPassword`, `ResetPassword`, `VerifyToken`, `GetOIDCAuthURL`, and `HandleOIDCCallback`.
- **Reusable Client & Middleware:** Any Go microservice can import `status-page/packages/auth/client` to verify tokens over gRPC.

### 3. `packages/core` (Hexagonal Domain & Storage Adapters)
- **Domain:** Models for `Status`, `FeatureStatus`, `CheckDefinition`, `CheckResult`, `StatusItemConfig`.
- **Port:** `StoragePort` interface decoupling storage from business logic.
- **Adapters:**
  - `sqlite`: Pure Go, CGO-free local database using `modernc.org/sqlite`.
  - `dynamodb`: AWS DynamoDB adapter via AWS SDK v2.
  - `s3`: AWS S3 JSON document storage adapter via AWS SDK v2.
- **Factory:** Instantiates the correct adapter at startup based on `STORAGE_ENGINE=sqlite|dynamodb|s3`.
- **Modular Checks Registry:** Extensible health check system with built-in `http`, `tcp`, `dns`, and `synthetic` checkers.

### 3. `packages/api` (Go REST API)
- Serves status data to the frontend:
  - `GET /health` — Service healthcheck
  - `GET /api/status` — All platform statuses
  - `GET /api/status/{tenant}` — Product statuses for a tenant
  - `GET /api/status/{tenant}/{product}` — Specific product status

### 4. `packages/status-probe-worker` (Background Evaluator)
- **Startup:** Validates `STATUS_DEF_JSON` (or `STATUS_DEF_FILE`) environment variable for correct formatting and required fields.
- **Sync:** Syncs unrecorded status records into the database.
- **Evaluation Loop:** On each loop iteration (configured via `WORKER_LOOP_INTERVAL`), finds statuses in DB with `last_update` older than `pull_interval_seconds`.
- **Concurrent Checks:** Concurrently executes the array of defined checks using modular check types, groups them into features, computes overall health, and updates the DB.

---

## 🚀 Quick Start

### 1. Run all tests
```bash
# Test all Go modules (core, api, worker)
go test status-page/packages/core/... status-page/packages/api/... status-page/packages/status-probe-worker/...

# Test Web frontend (Vitest)
npm --prefix packages/web test
```

### 2. Start the Eval Status Worker
```bash
# Uses pure Go SQLite by default with sample built-in status definitions
go run status-page/packages/status-probe-worker
```

### 3. Start the API Server
```bash
# Runs on :8080 by default
go run status-page/packages/api
```

### 4. Start the Web Frontend (SSR)
```bash
npm --prefix packages/web run dev
# Open http://localhost:3000 in your browser
```

---

## ⚙️ Configuration & Environment Variables

### Storage Engine Selection
| Variable | Values | Description |
|---|---|---|
| `STORAGE_ENGINE` | `sqlite` (default), `dynamodb`, `s3` | Selects the active Hexagonal storage adapter |
| `SQLITE_DSN` | e.g. `status.db` | Filepath or connection string for SQLite |
| `DYNAMODB_TABLE` | e.g. `status_records` | AWS DynamoDB table name |
| `DYNAMODB_ENDPOINT` | e.g. `http://localhost:8000` | Optional endpoint (e.g. for LocalStack / DynamoDB Local) |
| `S3_BUCKET` | e.g. `my-status-bucket` | AWS S3 bucket name |
| `S3_ENDPOINT` | e.g. `http://localhost:4566` | Optional endpoint (e.g. for LocalStack / MinIO) |
| `AWS_REGION` | e.g. `us-east-1` | AWS region |

### Status Worker Definition Format (`STATUS_DEF_JSON`)
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
        "id": "redis-check",
        "feature": "rate-limiter",
        "type": "tcp",
        "target": "redis.internal:6379",
        "timeout_ms": 1500
      }
    ]
  }
]
```
