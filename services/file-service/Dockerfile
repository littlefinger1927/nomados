# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copy shared packages first (referenced by go.mod replace directives)
COPY packages/shared-types/ /app/packages/shared-types/
COPY packages/logging/ /app/packages/logging/

# Copy service source (including go.mod and go.sum)
COPY services/file-service/ /app/services/file-service/

# Build from the service directory where go.mod lives
WORKDIR /app/services/file-service
RUN GOWORK=off go mod download
RUN GOWORK=off CGO_ENABLED=0 GOOS=linux go build -o /app/server ./cmd/main.go

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

EXPOSE 50056

CMD ["./server"]