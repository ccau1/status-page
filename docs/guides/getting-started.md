# Getting Started

This guide walks you through setting up and running the Status Page platform locally.

---

## 📋 Prerequisites

* **Go:** 1.24+ (1.25/1.26 recommended)
* **Node.js:** v20+ (v24 recommended) & npm
* **Docker & Docker Compose:** (Optional, for containerized execution)

---

## ⚡ Quick Start with Docker (Recommended)

The easiest way to spin up the entire ecosystem with hot-reloading:

```bash
# Start all services (Web, API, Worker) in the background
make up

# Inspect streaming container logs
make logs

# Stop all services and release network resources
make down
```

Once running:
* **Web Status Dashboard:** [http://localhost:3001](http://localhost:3001)
* **REST API:** [http://localhost:8085/api/status](http://localhost:8085/api/status)
* **API Health Check:** [http://localhost:8085/health](http://localhost:8085/health)

---

## 💻 Running Directly on the Host (Without Docker)

You can also run each component natively on your machine:

### 1. Install Web Dependencies
```bash
cd packages/web
npm install
```

### 2. Start the Background Worker
In a new terminal:
```bash
# Runs the evaluation worker using pure Go SQLite
go run status-page/packages/status-probe-worker
```

### 3. Start the REST API
In a second terminal:
```bash
# Starts the Go API on port 8080
PORT=8080 go run status-page/packages/api
```

### 4. Start the Vite SSR Web App
In a third terminal:
```bash
cd packages/web
npm run dev
# Dashboard available at http://localhost:3000
```
