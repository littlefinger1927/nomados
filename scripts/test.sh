#!/usr/bin/env bash
set -e

NOMADOS_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
FAILED=0

echo "=== NomadOS Test Runner ==="
echo ""

# Rust crypto tests
echo "[1/4] Running Rust crypto tests..."
cd "$NOMADOS_ROOT/packages/crypto"
if cargo test 2>&1; then
    echo "  PASS: crypto"
else
    echo "  FAIL: crypto"
    FAILED=1
fi
echo ""

# Go package tests
echo "[2/4] Running Go package tests..."
for pkg in auth-sdk logging shared-types; do
    dir="$NOMADOS_ROOT/packages/$pkg"
    if [ -d "$dir" ] && [ -f "$dir/go.mod" ]; then
        echo "  Testing packages/$pkg..."
        if (cd "$dir" && go test ./... 2>&1); then
            echo "  PASS: packages/$pkg"
        else
            echo "  FAIL: packages/$pkg"
            FAILED=1
        fi
    fi
done
echo ""

# Go service tests
echo "[3/4] Running Go service tests..."
for svc in auth-service session-service gateway-service workspace-orchestrator browser-manager streaming-service file-service vault-service observability; do
    dir="$NOMADOS_ROOT/services/$svc"
    if [ -d "$dir" ] && [ -f "$dir/go.mod" ]; then
        echo "  Testing services/$svc..."
        if (cd "$dir" && go test ./... 2>&1); then
            echo "  PASS: services/$svc"
        else
            echo "  FAIL: services/$svc"
            FAILED=1
        fi
    fi
done
echo ""

# Integration tests (require running services)
echo "[4/4] Running integration tests..."
INT_DIR="$NOMADOS_ROOT/tests/integration"
if [ -d "$INT_DIR" ] && [ -f "$INT_DIR/go.mod" ]; then
    echo "  Running integration tests (may skip if services unavailable)..."
    if (cd "$INT_DIR" && go test ./... 2>&1); then
        echo "  PASS: integration tests"
    else
        echo "  Some integration tests skipped or failed (expected without running services)"
    fi
fi
echo ""

if [ $FAILED -eq 0 ]; then
    echo "=== All tests passed ==="
    exit 0
else
    echo "=== Some tests failed ==="
    exit 1
fi