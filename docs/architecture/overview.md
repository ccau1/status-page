# Architecture Overview

The Platform Status Page is built around **Hexagonal Architecture** (Ports and Adapters) and a **Stateless Service-Oriented Model** to ensure resilience, multi-node horizontal scaling, and complete decoupling of domain logic, authentication, and persistence infrastructure.

---

## 🏛 System Topology

```text
               +-------------------------------------------------------+
               |                  Browser / Clients                    |
               +---------------------------+---------------------------+
                                           |
                                HTTP (SSR & Hydration)
                                           v
                             +---------------------------+
                             |       packages/web        |
                             |   (Vite SSR + React 19)   |
                             +-------------+-------------+
                                           |
                                       REST API
                                           v
                             +---------------------------+
                             |       packages/api        |
                             |       (Go HTTP Gateway)   |
                             +----+-----------------+----+
                                  |                 |
                        gRPC Calls (Port 50051)     Uses StoragePort Interface
                                  |                 |
                                  v                 v
                   +-------------------+   +--------------------+
                   |   packages/auth   |   |   packages/core    |
                   | (Stateless gRPC)  |   | (Domain & Adapters)|
                   +-------------------+   +----+---------------+
                                                |
                                                |
                             [ Pluggable Storage Adapters ]
                             +------------------+-------------------+
                             |                  |                   |
                       SQLite Adapter    DynamoDB Adapter       S3 Adapter
                             ^                  ^                   ^
                             |                  |                   |
                             +------------------+-------------------+
                                         Writes Evaluated Statuses
                                                ^
                                                |
                                  +-------------+-------------+
                                  | packages/status-probe-worker|
                                  |    (Go Polling Daemon)    |
                                  +---------------------------+
```

---

## 🎯 Key Architectural Pillars

### 1. Public vs. Internal Views
* **Public View (`/{tenant?}/{locale?}/status/{product?}`):**
  * Fast, 100% unauthenticated, and globally cacheable by CDNs or edge proxies.
  * Displays platform and product health, latency badges, 60-day uptime bars, and active announcement banners.
  * Feature drill-downs execute as instantaneous in-page client-side filters without network requests.
* **Internal Admin View (`/admin`):**
  * Incident Management Center for operations and on-call engineers.
  * Supports Dual-Mode Authentication: **Enterprise SSO (Microsoft Entra ID / Okta)** and **Username/Password Credentials**.
  * Allows publishing incident announcement banners, updating chronological investigation progress stages (`investigating` $\rightarrow$ `identified` $\rightarrow$ `monitoring` $\rightarrow$ `resolved`), and deactivating resolved banners.

### 2. Standalone Stateless gRPC Auth Service (`packages/auth`)
* The authentication service runs as a dedicated, standalone **gRPC microservice** listening on port `50051`.
* Handles all authentication RPCs: `Login`, `Register`, `Logout`, `ForgotPassword`, `ResetPassword`, `VerifyToken`, `GetOIDCAuthURL`, and `HandleOIDCCallback`.
* **Zero Database Dependencies:** Operates 100% statelessly using cryptographic HMAC-SHA256 signed JWTs and constant-time password comparisons.
* **gRPC Middleware in API:** All mutating administrative actions (`POST /api/incidents`, `DELETE /api/incidents/{id}`) are verified by invoking the auth service via gRPC before execution.

### 3. Multi-Node Stateless VPC EC2 Deployment
* **Horizontally Scalable:** Multiple stateless EC2 instances run `packages/web` and `packages/api` behind an AWS Application Load Balancer (ALB).
* **No Sticky Sessions Required:** Since JWT session cookies are cryptographically verified in-memory by any node, user sessions survive node termination and failover seamlessly.
* **Single Worker Replica:** A single background replica of `packages/status-probe-worker` continually evaluates health probes and writes snapshots to the shared database, preventing duplicate probe traffic against target systems.

### 4. Hexagonal Storage Ports & Adapters (`packages/core`)
* Business rules and data access are decoupled behind the `ports.StoragePort` contract.
* **DynamoDB (Multi-Node VPC):** Low-cost, serverless NoSQL storage connected via AWS VPC Gateway Endpoints with zero data-transfer fees.
* **SQLite on EBS (Single-Node / Local Dev):** Pure Go, CGO-free local database via `modernc.org/sqlite`.
