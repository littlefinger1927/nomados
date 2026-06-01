# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /workspace

# Copy workspace configuration
COPY go.work go.work.sum ./

# Copy all packages that streaming-service depends on
COPY packages/logging/ ./packages/logging/

# Copy service source
COPY services/streaming-service/ ./services/streaming-service/

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server ./services/streaming-service/cmd/main.go

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

EXPOSE 50055

CMD ["./server"]