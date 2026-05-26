#!/bin/bash
set -e

NOMADOS_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
COMPOSE_FILE="$NOMADOS_ROOT/infrastructure/docker/docker-compose.yml"

echo "=== NomadOS Development Environment ==="
echo ""

# Start infrastructure
echo "[1/4] Starting infrastructure (PostgreSQL, Redis, MinIO, NATS, TURN)..."
docker compose -f "$COMPOSE_FILE" up -d

echo "Waiting for infrastructure to be healthy..."
sleep 5

echo ""
echo "[2/4] Starting Go services..."

# Start each Go service in the background
start_service() {
    local name=$1
    local dir=$2
    shift 2

    if [ ! -d "$NOMADOS_ROOT/$dir" ]; then
        echo "  SKIP: $name (directory not found)"
        return
    fi

    echo "  Starting $name..."
    (cd "$NOMADOS_ROOT/$dir" && go run ./cmd/... "$@") &
    local pid=$!
    echo "  $name started (PID: $pid)"
}

# Services that need infrastructure
start_service "auth-service"        "services/auth-service"        &
start_service "session-service"     "services/session-service"     &
start_service "workspace-orchestrator" "services/workspace-orchestrator" &
start_service "browser-manager"      "services/browser-manager"    &
start_service "streaming-service"    "services/streaming-service"  &
start_service "file-service"         "services/file-service"       &
start_service "vault-service"        "services/vault-service"      &
start_service "observability"        "services/observability"      &

# Wait for Go services to start
sleep 3

echo ""
echo "[3/4] Starting gateway service..."
(cd "$NOMADOS_ROOT/services/gateway-service" && go run ./cmd/...) &
GATEWAY_PID=$!
echo "  gateway-service started (PID: $GATEWAY_PID)"

echo ""
echo "[4/4] Development environment ready."
echo ""
echo "NomadOS services are running:"
echo "  PostgreSQL:     localhost:5432"
echo "  Redis:          localhost:6379"
echo "  MinIO:          localhost:9000 (console: localhost:9001)"
echo "  NATS:           localhost:4222 (monitor: localhost:8222)"
echo "  TURN:           localhost:3478"
echo ""
echo "  Auth Service:       localhost:50051 (gRPC)"
echo "  Session Service:    localhost:50052 (gRPC)"
echo "  Workspace Orch:     localhost:50053 (gRPC)"
echo "  Browser Manager:   localhost:50054 (gRPC)"
echo "  Streaming Service:  localhost:50055 (gRPC)"
echo "  File Service:       localhost:50056 (gRPC)"
echo "  Vault Service:      localhost:50057 (gRPC)"
echo "  Gateway:            localhost:8080  (HTTP)"
echo "  Observability:      localhost:9090  (HTTP metrics)"
echo ""
echo "To stop all services:"
echo "  docker compose -f $COMPOSE_FILE down"
echo "  kill $GATEWAY_PID (and other Go service PIDs)"