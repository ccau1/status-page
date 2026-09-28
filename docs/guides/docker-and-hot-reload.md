# Docker & Hot Reload Setup

The repository is fully equipped with containerized development environments featuring live hot reloading across all four services.

---

## 🐳 Architecture of `docker-compose.yml`

```text
Host Machine                         Docker Engine
┌───────────────────────┐            ┌──────────────────────────────────────────┐
│ packages/web/src/     │ ──Bind──>  │ status-page-web (Node 24 / Vite SSR HMR) │
│ packages/api/         │ ──Bind──>  │ status-page-api (Air Go Hot-Reload)      │
│ packages/auth/        │ ──Bind──>  │ status-page-auth (Air gRPC Hot-Reload)   │
│ packages/probe-worker/│ ──Bind──>  │ status-page-probe-worker (Air Hot-Reload)│
│ packages/core/        │ ──Bind──>  │ Shared across Go containers              │
└───────────────────────┘            └────────────────────┬─────────────────────┘
                                                          │
                                         Internal gRPC Network (auth:50051)
                                         & Shared Volume (/data/status.db)
                                                          v
                                             ┌──────────────────────────┐
                                             │ status-page-data         │
                                             │ (/data/status.db)        │
                                             └──────────────────────────┘
```

---

## 🔥 How Hot Reloading Works

### 1. Go Services (`api`, `auth`, `status-probe-worker`)
* **Tool:** [Air](https://github.com/air-verse/air).
* **Configurations:** `packages/api/.air.toml`, `packages/auth/.air.toml`, and `packages/status-probe-worker/.air.toml`.
* **Behavior:** Air monitors `.go` files in the service directory and in `packages/core/`. When an edit is saved, Air triggers an incremental rebuild and transparently restarts the binary in under 500ms.
* **Volume Performance:** Air writes binaries to the container's native `/tmp/` directory rather than the host-mounted filesystem to bypass cross-OS VirtioFS filesystem latency on macOS and Windows.

### 2. Frontend Web App (`packages/web`)
* **Tool:** Vite HMR (Hot Module Replacement).
* **Configuration:** `packages/web/vite.config.ts`.
* **Polling:** Configured with `watch: { usePolling: true }` and `host: '0.0.0.0'` so file system notifications reliably propagate from host operating systems into the Alpine Linux container.

---

## 🛠 Makefile Lifecycle Commands

| Command | Action |
|---|---|
| `make up` | Builds all 4 containers and starts the stack in the background |
| `make down` | Gracefully terminates and removes containers, networks, and volumes |
| `make restart` | Restarts the complete environment (`make down && make up`) |
| `make logs` | Tails live streaming logs from all four containers simultaneously |
| `make test` | Executes the complete test suite across Go modules and Vitest |
| `make build` | Compiles production binaries and SSR frontend bundles |
| `make clean` | Removes local build outputs, temporary files, and database files |
