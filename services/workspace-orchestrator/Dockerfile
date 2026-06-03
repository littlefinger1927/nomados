# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copy shared packages first (referenced by go.mod replace directives)
COPY packages/shared-types/ /app/packages/shared-types/
COPY packages/logging/ /app/packages/logging/

# Copy service source (including go.mod and go.sum)
COPY services/workspace-orchestrator/ /app/services/workspace-orchestrator/

# Build from the service directory where go.mod lives
WORKDIR /app/services/workspace-orchestrator
RUN GOWORK=off CGO_ENABLED=0 GOOS=linux go build -o /app/server ./cmd/main.go

# Runtime stage
FROM alpine:3.19

# Install Docker CLI for container management
RUN apk --no-cache add ca-certificates tzdata docker-cli

# Docker socket access required for container management
# TODO: add appuser to docker group by matching host GID for production
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/server .

EXPOSE 50053

CMD ["./server"]