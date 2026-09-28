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
RUN cd packages/auth && go mod download
RUN cd packages/api && go mod download

# Copy source trees
COPY packages/core ./packages/core
COPY packages/auth ./packages/auth
COPY packages/api ./packages/api

# Pre-warm Go build cache for lightning fast hot reload starts
RUN go build -o /tmp/air-api/api-main status-page/packages/api

WORKDIR /app

EXPOSE 8080

CMD ["air", "-c", "packages/api/.air.toml"]

# -----------------------------------------------------------------------------
# Production build stage
# -----------------------------------------------------------------------------
FROM golang:1.26-alpine AS builder

WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY go.work ./
COPY packages/core ./packages/core
COPY packages/auth ./packages/auth
COPY packages/api ./packages/api
COPY packages/status-probe-worker ./packages/status-probe-worker

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-w -s" -o /bin/api status-page/packages/api

# Final minimal production image
FROM alpine:3.21 AS prod

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /bin/api /app/api

EXPOSE 8080

CMD ["/app/api"]
