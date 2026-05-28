# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /workspace

# Copy workspace configuration
COPY go.work go.work.sum ./

# Copy all packages that observability depends on
COPY packages/shared-types/ ./packages/shared-types/
COPY packages/logging/ ./packages/logging/

# Copy service source
COPY services/observability/ ./services/observability/

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server ./services/observability/cmd/main.go

# Runtime stage
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN adduser -D -g '' appuser

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/server .

# Use non-root user
USER appuser

EXPOSE 9090

CMD ["./server"]