# Platform Status Documentation

Welcome to the comprehensive technical documentation for the Status Page platform. This repository hosts a distributed, multi-tenant status monitoring and reporting engine consisting of a Vite SSR frontend, Go Hexagonal API, background evaluation worker, and a shared hexagonal core.

---

## 📚 Documentation Index

### 1. [Architecture](./architecture/)
Deep-dive into the architectural patterns, data flow, and design principles:
* **[System Overview](./architecture/overview.md)** — High-level monorepo topology, data flow, and service boundaries.
* **[Hexagonal Storage & Adapters](./architecture/storage-ports-and-adapters.md)** — Decoupled persistence layer supporting SQLite, AWS DynamoDB, and AWS S3.
* **[Status Probe Worker Pipeline](./architecture/probe-worker-pipeline.md)** — Polling mechanics, ENV JSON validation, and state aggregation.
* **[Modular Check System](./architecture/modular-checks.md)** — Extensible health-check architecture, built-in types (`http`, `tcp`, `dns`, `synthetic`), and adding custom checkers.

### 2. [Packages](./packages/)
Detailed references and specifications for each monorepo package:
* **[`packages/web`](./packages/web.md)** — Vite SSR React application, dynamic URL routing (`/{tenant?}/{locale?}/status/{product?}`), in-page feature filtering, and Obsidian glassmorphism styling.
* **[`packages/auth`](./packages/auth.md)** — Reusable enterprise SSO (Microsoft AD / Okta) and stateless JWT verification module.
* **[`packages/api`](./packages/api.md)** — Go REST API, port injection, CORS, response envelopes, and endpoint contracts.
* **[`packages/core`](./packages/core.md)** — Shared hexagonal core containing domain models, storage ports, adapters, and check registries.
* **[`packages/status-probe-worker`](./packages/status-probe-worker.md)** — Background Go evaluation daemon, interval loops, and status syncing.

### 3. [Guides & Operations](./guides/)
Step-by-step guides for development, configuration, and verification:
* **[Getting Started](./guides/getting-started.md)** — Prerequisites, quick setup, and running services locally.
* **[Docker & Hot Reload](./guides/docker-and-hot-reload.md)** — Docker Compose architecture, Air for Go hot-reloading, Vite polling HMR, and Makefile commands.
* **[Configuration & Environment Variables](./guides/configuration.md)** — Complete reference of storage and check configuration options.
* **[Testing & Quality Assurance](./guides/testing-and-ci.md)** — Running unit, integration, and component tests across Go and TypeScript.
