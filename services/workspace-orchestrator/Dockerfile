# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /workspace

# Copy workspace configuration
COPY go.work go.work.sum ./

# Copy all packages that workspace-orchestrator depends on
COPY packages/shared-types/ ./packages/shared-types/
COPY packages/logging/ ./packages/logging/

# Copy service source
COPY services/workspace-orchestrator/ ./services/workspace-orchestrator/

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server ./services/workspace-orchestrator/cmd/main.go

# Runtime stage
FROM alpine:3.19

# Install Docker CLI for container management
RUN apk --no-cache add ca-certificates tzdata docker-cli

# Create non-root user
RUN adduser -D -g '' appuser

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/server .

# Use non-root user
USER appuser

EXPOSE 50053

CMD ["./server"]