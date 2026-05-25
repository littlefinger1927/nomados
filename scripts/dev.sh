#!/bin/bash
set -e

NOMADOS_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
COMPOSE_FILE="$NOMADOS_ROOT/infrastructure/docker/docker-compose.yml"

echo "Starting NomadOS development infrastructure..."
docker compose -f "$COMPOSE_FILE" up -d

echo "Waiting for services to be healthy..."
sleep 5

echo ""
echo "NomadOS infrastructure is running:"
echo "  PostgreSQL:  localhost:5432"
echo "  Redis:       localhost:6379"
echo "  MinIO:       localhost:9000 (console: localhost:9001)"
echo "  NATS:        localhost:4222 (monitor: localhost:8222)"
echo "  TURN:        localhost:3478"
echo ""
echo "To stop: docker compose -f $COMPOSE_FILE down"