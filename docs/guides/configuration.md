# Configuration Reference

This document details all environment variables and configuration structures supported by the platform.

---

## 📄 Using `.env` and `.env.example`

A template file **`.env.example`** is provided in the repository root. To configure your local development or production environment:

```bash
# Copy template to .env
cp .env.example .env

# Edit .env to customize status checks, pull intervals, or storage engines
$EDITOR .env
```

Both Docker Compose and local Go binaries (`packages/api`, `packages/status-probe-worker`) automatically parse and load values from `.env` on startup.

---

## 🗄 Storage Engine Settings

Controlled via the `STORAGE_ENGINE` variable:

| Variable | Engine | Default | Description |
|---|---|---|---|
| `STORAGE_ENGINE` | All | `sqlite` | Selected storage adapter: `sqlite`, `dynamodb`, or `s3` |
| `SQLITE_DSN` | `sqlite` | `status.db` | File path or connection string for SQLite database |
| `DYNAMODB_TABLE` | `dynamodb` | `status_records` | AWS DynamoDB table name |
| `DYNAMODB_ENDPOINT` | `dynamodb` | *(empty)* | Custom endpoint for LocalStack or DynamoDB Local |
| `S3_BUCKET` | `s3` | `status-page-bucket` | AWS S3 bucket name |
| `S3_ENDPOINT` | `s3` | *(empty)* | Custom endpoint for LocalStack or MinIO |
| `AWS_REGION` | AWS | `us-east-1` | AWS region |

---

## ⚙️ Worker Daemon Settings

| Variable | Default | Description |
|---|---|---|
| `WORKER_LOOP_INTERVAL` | `5s` | Sleep duration between evaluation loop iterations (e.g. `2s`, `10s`, `1m`) |
| `STATUS_DEF_JSON` | *(built-in defaults)* | Raw JSON string defining all products, pull intervals, and health checks |
| `STATUS_DEF_FILE` | *(empty)* | Path to a local `.json` file containing the status check configurations |

### Health Check Definition Fields

Each entry in `STATUS_DEF_JSON` groups checks under a `tenant` + `product`. A check supports:

| Field | Required | Description |
|---|---|---|
| `id` | ✅ | Unique check identifier |
| `feature` | ✅ | Feature key this check validates; multiple checks can share one feature |
| `region` | optional | Region tag (e.g. `us-east-1`). Checks with a region also feed a per-region feature breakdown (`regional_features` in the status payload), which powers the region filter on the status page. Checks without a region only feed the global rollup |
| `type` | ✅ | Check method: `http`, `tcp`, `synthetic`, etc. |
| `target` | ✅ | URL or `host:port` to probe |
| `timeout_ms` | optional | Per-check timeout override |
| `parameters` | optional | Type-specific options (e.g. `simulated_state` for `synthetic`) |

The global feature state is always the worst case across all its checks, so a regional outage still degrades the product-level view.

### Tenant Templates (Document Form)

When many tenants monitor the same products, `STATUS_DEF_JSON` (or `STATUS_DEF_FILE`) can use the **document form** instead of a bare array. Define each product+check set once under `templates`, then bind tenants to it under `items`:

```json
{
  "templates": {
    "standard-billing": {
      "product": "billing-engine",
      "pull_interval_seconds": 20,
      "checks": [
        {
          "id": "tax-check",
          "feature": "tax-calc",
          "region": "{{region}}",
          "type": "tcp",
          "target": "tax.{{tenant}}.cloud.internal:443"
        }
      ]
    }
  },
  "items": [
    { "tenant": "acme", "template": "standard-billing",
      "variables": { "region": "us-east-1" } },
    { "tenant": "globex", "template": "standard-billing",
      "pull_interval_seconds": 60,
      "variables": { "region": "eu-west-1" },
      "check_overrides": [
        { "id": "tax-check", "timeout_ms": 8000 },
        { "id": "fraud-check", "feature": "fraud-scan", "type": "tcp",
          "target": "fraud.globex.internal:9000" },
        { "id": "stripe-check", "remove": true }
      ]
    }
  ]
}
```

Rules:

* **Variables** — `{{tenant}}` and `{{product}}` are built in; `items[].variables` adds more (and wins on collision). Substitution applies to all check string fields and `parameters` values, including on inline (template-less) items. An unresolved `{{var}}` fails startup with a descriptive error.
* **Scalar overrides** — `product` / `pull_interval_seconds` on an item override the template's values.
* **`check_overrides`** merge by check `id`, applied in order: matching IDs are field-merged (non-empty fields win; `parameters` merge per-key), unknown IDs are appended as new checks (must be fully defined), and `"remove": true` drops a check (error if the ID doesn't exist).
* Items without a `template` behave exactly like legacy array entries. The legacy bare-array form keeps working unchanged — both forms are detected automatically.
* Expansion happens once at worker startup; templates themselves are never validated until expanded per tenant.

---

## 🌐 Web & API Server Settings

| Variable | Service | Default | Description |
|---|---|---|---|
| `PORT` | API | `8080` | Port for the Go REST API |
| `PORT` | Web | `3000` | Port for the Node.js / Vite SSR server |
| `API_BASE_URL` | Web | `http://localhost:8080` | Base URL used by the web application to fetch status updates |
| `API_PORT` | Docker | `8085` | Host port mapped to API container |
| `WEB_PORT` | Docker | `3001` | Host port mapped to Web container |
| `AUTH_GRPC_ADDR` | API | `localhost:50051` | Address of the standalone Auth gRPC service |
| `INCIDENT_BANNER_JSON` | API | *(empty)* | JSON defining active operator announcement banners with progress timeline |

---

## 🔐 Auth Service & gRPC Settings

| Variable | Service | Default | Description |
|---|---|---|---|
| `AUTH_GRPC_PORT` | Auth | `50051` | Port on which the standalone Auth gRPC daemon listens |
| `PASSWORD_AUTH_ENABLED`| Auth | `true` | Enables/disables username and password authentication |
| `ADMIN_USER` | Auth | `admin` | Default local administrator username |
| `ADMIN_PASSWORD` | Auth | `admin` | Default local administrator password (change in production!) |
| `LOCAL_USERS` | Auth | *(empty)* | Comma-separated list of secondary users (`user1:pass1,user2:pass2`) |
| `SSO_PROVIDER` | Auth | `mock` | `oidc` (production AD/Okta) or `mock` (local dev) |
| `SSO_PROVIDER_NAME` | Auth | `Local Dev SSO` | Display label for SSO button on `/admin` login screen |
| `OIDC_ISSUER_URL` | Auth | *(empty)* | Discovery URL for Microsoft Entra ID or Okta |
| `OIDC_CLIENT_ID` | Auth | *(empty)* | Enterprise Application Client ID |
| `OIDC_CLIENT_SECRET` | Auth | *(empty)* | Enterprise Application Client Secret |
| `OIDC_REDIRECT_URL` | Auth | `http://localhost:8085/auth/callback` | Authorized redirect URI for SSO callback |
| `SESSION_SECRET` | Auth | *(random)* | HMAC-SHA256 secret key for signing stateless JWT session cookies |

---

## 📢 Incident Banners & Check-Down Messaging

### 1. Operator-Controlled Announcement Banners
You can publish, update, and manage incident announcements across the top of the page in two ways:

#### A. Via `.env` (`INCIDENT_BANNER_JSON`):
```json
[
  {
    "id": "inc-001",
    "tenant": "*",
    "products": ["billing-engine"],
    "check_ids": ["tax-jurisdiction-check-eu"],
    "features": ["tax-calc"],
    "title": "Elevated Latency in Tax Calculation Service",
    "severity": "minor",
    "state": "monitoring",
    "message": "We have rerouted traffic to backup providers and are monitoring transaction queues.",
    "active": true,
    "updates": [
      {
        "timestamp": "2026-09-27T10:00:00Z",
        "state": "investigating",
        "message": "Engineers received automated alerts for high response latency."
      },
      {
        "timestamp": "2026-09-27T10:20:00Z",
        "state": "identified",
        "message": "Identified latency spike originating from third-party provider."
      }
    ]
  }
]
```

#### B. Via REST API (`POST /api/incidents`):
```bash
curl -X POST http://localhost:8085/api/incidents \
  -H "Content-Type: application/json" \
  -d '{
    "id": "inc-002",
    "tenant": "*",
    "title": "Scheduled Database Maintenance",
    "severity": "maintenance",
    "state": "monitoring",
    "message": "Running non-blocking database index rebuilds between 02:00 - 04:00 UTC.",
    "active": true
  }'
```

### 2. Automated Pre-Made Check-Down Messaging
When individual health checks fail and there is no active manual announcement banner:
* The system automatically scans for degraded/outage features across active products.
* Formulates a dynamic top banner message:
  * **Outages:** *"Automated checks reported failing responses on [Feature Name] ([Product]). Incident response procedures are actively engaged."*
  * **Degraded:** *"Automated health probes detected elevated response times on [Feature Name] ([Product]). Core traffic remains online while systems stabilize."*

---

## 📋 Complete `STATUS_DEF_JSON` Example

```json
[
  {
    "tenant": "enterprise",
    "product": "core-api",
    "pull_interval_seconds": 10,
    "checks": [
      {
        "id": "gateway-health",
        "feature": "api-gateway",
        "type": "http",
        "target": "https://api.company.internal/health",
        "timeout_ms": 3000,
        "expected_status": 200,
        "expected_body": "OK"
      },
      {
        "id": "db-pool",
        "feature": "database-cluster",
        "type": "tcp",
        "target": "db.internal:5432",
        "timeout_ms": 2000
      },
      {
        "id": "edge-dns",
        "feature": "dns-routing",
        "type": "dns",
        "target": "api.company.com"
      }
    ]
  }
]
```
