#!/usr/bin/env bash
# NomadOS Database Migration Script
# Usage:
#   ./migrate.sh up          - Apply all pending migrations
#   ./migrate.sh down [N]    - Roll back N migrations (default: 1)
#   ./migrate.sh version     - Show current migration version
#   ./migrate.sh force [V]   - Force migration version (for fixing dirty state)

set -uo pipefail

# Default DATABASE_URL (matches docker-compose dev setup)
DATABASE_URL="${DATABASE_URL:-postgres://nomados:nomados_dev@localhost:5432/nomados?sslmode=disable}"

# Project root (script lives in infrastructure/scripts/)
PROJECT_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"

# Migration directories and their custom migration table names.
# Each service gets its own schema_migrations table to avoid collisions
# when multiple services share the same database.
MIGRATION_DIRS=(
  "services/auth-service/migrations"
  "services/workspace-orchestrator/migrations"
  "services/file-service/migrations"
)
MIGRATION_TABLES=(
  "schema_migrations_auth"
  "schema_migrations_workspace"
  "schema_migrations_file"
)

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m'

info()  { echo -e "${GREEN}[migrate]${NC} $1"; }
warn()  { echo -e "${YELLOW}[migrate]${NC} $1" >&2; }
error() { echo -e "${RED}[migrate]${NC} $1" >&2; exit 1; }

# Check for golang-migrate
check_migrate() {
  if ! command -v migrate &>/dev/null; then
    error "migrate CLI not found. Install with:
  macOS:  brew install golang-migrate
  Linux:  curl -L https://github.com/golang-migrate/migrate/releases/latest/download/migrate.linux-amd64.tar.gz | tar xvz
  Go:     go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"
  fi
}

# Build a database URL with a custom migrations table
db_url_with_table() {
  local table="$1"
  local base_url="$DATABASE_URL"
  # Append x-migrations-table as a query parameter
  if [[ "$base_url" == *"?"* ]]; then
    echo "${base_url}&x-migrations-table=${table}"
  else
    echo "${base_url}?x-migrations-table=${table}"
  fi
}

run_for_each_dir() {
  local cmd="$1"
  local reverse="${2:-false}"
  shift 2
  local dirs=("${MIGRATION_DIRS[@]}")
  local tables=("${MIGRATION_TABLES[@]}")
  if [ "$reverse" = "true" ]; then
    local dirs_rev=()
    local tables_rev=()
    for ((i=${#dirs[@]}-1; i>=0; i--)); do
      dirs_rev+=("${dirs[$i]}")
      tables_rev+=("${tables[$i]}")
    done
    dirs=("${dirs_rev[@]}")
    tables=("${tables_rev[@]}")
  fi
  local idx=0
  for dir in "${dirs[@]}"; do
    local abs_dir="${PROJECT_ROOT}/${dir}"
    local service_name
    service_name="$(basename "$(dirname "$dir")")"
    local db_url
    db_url="$(db_url_with_table "${tables[$idx]}")"
    if [ -d "$abs_dir" ]; then
      info "$service_name: $cmd"
      migrate -database "$db_url" -source "file://${abs_dir}" $cmd "$@" 2>&1 || warn "$service_name: $cmd failed (may already be applied)"
    else
      warn "Migration directory not found: $abs_dir"
    fi
    idx=$((idx + 1))
  done
}

case "${1:-help}" in
  up)
    check_migrate
    run_for_each_dir "up" "false"
    info "All migrations applied"
    ;;

  down)
    check_migrate
    steps="${2:-1}"
    run_for_each_dir "down" "true" "$steps"
    info "Rollback complete"
    ;;

  version)
    check_migrate
    run_for_each_dir "version" "false"
    ;;

  force)
    check_migrate
    version="${2:?Version number required}"
    warn "Forcing migration version to $version"
    run_for_each_dir "force" "false" "$version"
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
    echo "  force [V]   Force migration version (for fixing dirty state)"
    echo ""
    echo "Environment:"
    echo "  DATABASE_URL  PostgreSQL connection string"
    echo "                Default: postgres://nomados:nomados_dev@localhost:5432/nomados?sslmode=disable"
    ;;
esac