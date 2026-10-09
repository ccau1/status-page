.PHONY: up down restart logs build test clean

API_PORT ?= 8085
WEB_PORT ?= 3001
AUTH_GRPC_PORT ?= 50051

# Spin up all services with hot reload
up:
	AUTH_GRPC_PORT=$(AUTH_GRPC_PORT) API_PORT=$(API_PORT) WEB_PORT=$(WEB_PORT) docker compose up --build -d
	@echo "Services are up!"
	@echo "Web frontend: http://localhost:$(WEB_PORT)"
	@echo "REST API:     http://localhost:$(API_PORT)"
	@echo "Auth gRPC:    localhost:$(AUTH_GRPC_PORT)"
	@echo "Run 'make logs' to view streaming container logs."

# Spin down all services and network
down:
	docker compose down
	@echo "All services stopped."

# Restart all services
restart: down up

# Stream container logs
logs:
	docker compose logs -f

# Run tests across all packages
test:
	@echo "Running Go tests..."
	go test status-page/packages/core/... status-page/packages/auth/... status-page/packages/api/... status-page/packages/status-probe-worker/...
	@echo "Running Playwright Runner tests..."
	npm --prefix packages/playwright-runner test
	@echo "Running Web frontend tests..."
	npm --prefix packages/web test

# Build production binaries and assets locally
build:
	@echo "Building Go API & Worker..."
	cd packages/api && go build -o ../../bin/api .
	cd packages/status-probe-worker && go build -o ../../bin/worker .
	@echo "Building Playwright Runner..."
	npm --prefix packages/playwright-runner run build
	@echo "Building Web frontend..."
	npm --prefix packages/web run build

# Clean temporary files, build artifacts, and databases
clean:
	rm -rf bin/ tmp/ packages/*/tmp dist/ packages/web/dist *.db *.db-journal
