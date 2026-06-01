#!/usr/bin/env bash
# NomadOS Database Migration Script
# Usage:
#   ./migrate.sh up          - Apply all pending migrations
#   ./migrate.sh down [N]    - Roll back N migrations (default: 1)
#   ./migrate.sh version     - Show current migration version
#   ./migrate.sh force [V]   - Force migration version (for fixing dirty state)

set -euo pipefail

# Default DATABASE_URL (matches docker-compose dev setup)
DATABASE_URL="${DATABASE_URL:-postgres://nomados:nomados_dev@localhost:5432/nomados?sslmode=disable}"

# Migration directories (in order)
MIGRATION_DIRS=(
  "services/auth-service/migrations"
  "services/workspace-orchestrator/migrations"
  "services/file-service/migrations"
)

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m'

info()  { echo -e "${GREEN}[migrate]${NC} $1"; }
warn()  { echo -e "${YELLOW}[migrate]${NC} $1"; }
error() { echo -e "${RED}[migrate]${NC} $1" >&2; exit 1; }

# Check for golang-migrate
check_migrate() {
  if ! command -v migrate &>/dev/null; then
    error "migrate CLI not found. Install with:
  macOS:  brew install golang-migrate
  Linux:  curl -L https://github.com/golang-migrate/migrate/releases/latest/download/migrate.linux-amd64.tar.gz | tar xvz
  Go:     go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest"
  fi
}

# Build migration source path from directories
build_source() {
  local dirs=()
  for dir in "${MIGRATION_DIRS[@]}"; do
    local abs_dir
    abs_dir="$(cd "$(dirname "$0")/.." && pwd)/${dir}"
    if [ -d "$abs_dir" ]; then
      dirs+=("file://${abs_dir}")
    else
      warn "Migration directory not found: $abs_dir"
    fi
  done

  # golang-migrate supports multiple sources with comma separation
  local source=""
  for dir in "${dirs[@]}"; do
    if [ -n "$source" ]; then
      source="${source},${dir}"
    else
      source="$dir"
    fi
  done
  echo "$source"
}

case "${1:-help}" in
  up)
    check_migrate
    source=$(build_source)
    info "Running migrations up from: $source"
    migrate -database "$DATABASE_URL" -source "$source" up
    info "All migrations applied"
    ;;

  down)
    check_migrate
    source=$(build_source)
    steps="${2:-1}"
    info "Rolling back $steps migration(s)"
    migrate -database "$DATABASE_URL" -source "$source" down "$steps"
    info "Rollback complete"
    ;;

  version)
    check_migrate
    source=$(build_source)
    migrate -database "$DATABASE_URL" -source "$source" version
    ;;

  force)
    check_migrate
    source=$(build_source)
    version="${2:?Version number required}"
    warn "Forcing migration version to $version"
    migrate -database "$DATABASE_URL" -source "$source" force "$version"
    info "Version forced to $version"
    ;;

  help|*)
    echo "NomadOS Database Migration Script"
    echo ""
    echo "Usage: $0 {up|down|version|force} [args]"
    echo ""
    echo "Commands:"
    echo "  up          Apply all pending migrations"
    echo "  down [N]    Roll back N migrations (default: 1)"
    echo "  version     Show current migration version"
    echo "  force [V]   Force migration version (for dirty state)"
    echo ""
    echo "Environment:"
    echo "  DATABASE_URL  PostgreSQL connection string"
    echo "                Default: postgres://nomados:nomados_dev@localhost:5432/nomados?sslmode=disable"
    ;;
esac