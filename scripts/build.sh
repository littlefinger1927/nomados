#!/usr/bin/env bash
set -e

NOMADOS_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
FAILED=0

echo "=== NomadOS Build Script ==="
echo ""

# Rust crypto library
echo "[1/4] Building Rust crypto package..."
cd "$NOMADOS_ROOT/packages/crypto"
if cargo build --release 2>&1; then
    echo "  PASS: crypto (release)"
else
    echo "  FAIL: crypto"
    FAILED=1
fi
echo ""

# Go packages
echo "[2/4] Building Go packages..."
for pkg in auth-sdk logging shared-types; do
    dir="$NOMADOS_ROOT/packages/$pkg"
    if [ -d "$dir" ] && [ -f "$dir/go.mod" ]; then
        echo "  Building packages/$pkg..."
        if (cd "$dir" && go build ./... 2>&1); then
            echo "  PASS: packages/$pkg"
        else
            echo "  FAIL: packages/$pkg"
            FAILED=1
        fi
    fi
done
echo ""

# Go services
echo "[3/4] Building Go services..."
for svc in auth-service session-service gateway-service workspace-orchestrator browser-manager streaming-service file-service vault-service observability; do
    dir="$NOMADOS_ROOT/services/$svc"
    if [ -d "$dir" ] && [ -f "$dir/go.mod" ]; then
        echo "  Building services/$svc..."
        if (cd "$dir" && go build ./... 2>&1); then
            echo "  PASS: services/$svc"
        else
            echo "  FAIL: services/$svc"
            FAILED=1
        fi
    fi
done
echo ""

# Tauri client
echo "[4/4] Building Tauri client..."
TAURI_DIR="$NOMADOS_ROOT/apps/tauri-client/src-tauri"
if [ -d "$TAURI_DIR" ] && [ -f "$TAURI_DIR/Cargo.toml" ]; then
    if (cd "$TAURI_DIR" && cargo build 2>&1); then
        echo "  PASS: tauri-client"
    else
        echo "  FAIL: tauri-client (may need Tauri CLI installed)"
        FAILED=1
    fi
else
    echo "  SKIP: tauri-client (not found)"
fi
echo ""

if [ $FAILED -eq 0 ]; then
    echo "=== Build complete ==="
    echo ""
    echo "Binaries:"
    echo "  Go services: built in place (run with 'go run ./cmd/...')"
    echo "  Rust crypto: packages/crypto/target/release/"
    echo "  Tauri client: apps/tauri-client/src-tauri/target/"
    exit 0
else
    echo "=== Build failed ==="
    exit 1
fi