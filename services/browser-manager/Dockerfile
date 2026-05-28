# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /workspace

# Copy workspace configuration
COPY go.work go.work.sum ./

# Copy all packages that browser-manager depends on
COPY packages/logging/ ./packages/logging/

# Copy service source
COPY services/browser-manager/ ./services/browser-manager/

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server ./services/browser-manager/cmd/main.go

# Runtime stage
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata chromium

# Create non-root user with a home directory for browser profiles
RUN adduser -D -g '' -h /home/appuser appuser

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/server .

# Use non-root user
USER appuser

EXPOSE 50054

CMD ["./server"]