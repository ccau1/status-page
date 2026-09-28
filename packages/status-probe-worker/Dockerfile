# syntax=docker/dockerfile:1

# -----------------------------------------------------------------------------
# Development stage with hot reload (Air)
# -----------------------------------------------------------------------------
FROM golang:1.26-alpine AS dev

WORKDIR /app

RUN apk add --no-cache git curl build-base

# Install Air for hot-reload
RUN curl -sSfL https://raw.githubusercontent.com/air-verse/air/master/install.sh | sh -s -- -b /usr/local/bin

# Copy go workspace configuration
COPY go.work ./
COPY packages/core/go.mod packages/core/go.sum* ./packages/core/
COPY packages/auth/go.mod packages/auth/go.sum* ./packages/auth/
COPY packages/api/go.mod packages/api/go.sum* ./packages/api/
COPY packages/status-probe-worker/go.mod packages/status-probe-worker/go.sum* ./packages/status-probe-worker/

# Download Go module dependencies
RUN cd packages/core && go mod download
RUN cd packages/status-probe-worker && go mod download

# Copy source trees
COPY packages/core ./packages/core
COPY packages/status-probe-worker ./packages/status-probe-worker

# Pre-warm Go build cache for lightning fast hot reload starts
RUN go build -o /tmp/air-worker/worker-main status-page/packages/status-probe-worker

WORKDIR /app

CMD ["air", "-c", "packages/status-probe-worker/.air.toml"]

# -----------------------------------------------------------------------------
# Production build stage
# -----------------------------------------------------------------------------
FROM golang:1.26-alpine AS builder

WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY go.work ./
COPY packages/core ./packages/core
COPY packages/api ./packages/api
COPY packages/status-probe-worker ./packages/status-probe-worker

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-w -s" -o /bin/worker status-page/packages/status-probe-worker

# Final minimal production image
FROM alpine:3.21 AS prod

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /bin/worker /app/worker

CMD ["/app/worker"]
