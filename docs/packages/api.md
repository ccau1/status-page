# `packages/api` — REST API Backend

The `api` package is a Go HTTP REST service implementing Hexagonal Architecture. It connects to `packages/core` for persistent storage and acts as an HTTP gateway to `packages/auth` over gRPC.

---

## 📡 Public Read Endpoints

### 1. `GET /health`
Returns container vitality.
* **Status:** `200 OK`
* **Response:** `{ "status": "healthy" }`

### 2. `GET /api/status`
Returns all product status records across all tenants.

### 3. `GET /api/status/{tenant}`
Returns all product status records scoped to the requested tenant.

### 4. `GET /api/status/{tenant}/{product}`
Returns status and full feature breakdown for a specific product.

### 5. `GET /api/incidents` & `GET /api/incidents/{tenant}`
Returns active incident announcement banners and progress timelines.

---

## 🔐 Authentication Gateway Endpoints (Delegated to gRPC Auth Service)

The API forwards all incoming web authentication requests to `packages/auth` over gRPC:

| Endpoint | Method | gRPC Method Called | Description |
|---|---|---|---|
| `/auth/config` | `GET` | In-memory config | Returns provider name and password auth availability |
| `/auth/login` | `POST` | `auth.Login` | Validates credentials via gRPC, sets `status_session` cookie |
| `/auth/register` | `POST` | `auth.Register` | Registers operator account via gRPC, sets session cookie |
| `/auth/logout` | `POST` | `auth.Logout` | Clears `status_session` cookie |
| `/auth/forgot-password`| `POST` | `auth.ForgotPassword` | Generates signed 15-minute reset token via gRPC |
| `/auth/reset-password` | `POST` | `auth.ResetPassword` | Verifies reset token and updates password via gRPC |
| `/auth/me` | `GET` | `auth.VerifyToken` | Verifies current session cookie via gRPC, returns identity |
| `/auth/login/sso` | `GET` | `auth.GetOIDCAuthURL` | Redirects to Microsoft AD / Okta SSO login |
| `/auth/callback` | `GET` | `auth.HandleOIDCCallback` | Exchanges OIDC code for claims, sets session cookie |

---

## 🛡 Secure Admin Endpoints (Guarded by gRPC Middleware)

Administrative endpoints that modify platform incidents are guarded by `client.RequireAdmin(authClient, "status_session")`. The middleware extracts the session token and invokes `authClient.VerifyToken` over gRPC before permitting request execution:

### 1. `POST /api/incidents`
Publishes a new incident announcement banner or adds a timeline progress update.
* **Headers:** `Cookie: status_session=...` or `Authorization: Bearer <token>`
* **Request Body:**
  ```json
  {
    "id": "inc-101",
    "tenant": "*",
    "product": "billing-engine",
    "title": "Elevated Checkout Latency",
    "severity": "minor",
    "state": "monitoring",
    "message": "Traffic rerouted to secondary gateway. Monitoring transaction queues.",
    "active": true,
    "updates": [
      {
        "timestamp": "2026-09-27T10:00:00Z",
        "state": "investigating",
        "message": "Elevated latency detected."
      }
    ]
  }
  ```
* **Status:** `200 OK` (or `401 Unauthorized` / `403 Forbidden` if unverified)

### 2. `DELETE /api/incidents/{id}`
Deactivates and archives an incident announcement banner from public dashboards.
* **Status:** `200 OK`

---

## 🔒 Cross-Origin Resource Sharing (CORS)

Built-in CORS middleware allows local Vite frontend development and cross-domain embedding:
* `Access-Control-Allow-Origin: *`
* `Access-Control-Allow-Methods: GET, POST, DELETE, OPTIONS`
* Handles `OPTIONS` preflight requests cleanly with `200 OK`.
