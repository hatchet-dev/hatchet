#!/bin/bash
# build_all.sh - Build all Docker images for current branch

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
REPO_ROOT="$(cd "$TEST_DIR/../../.." && pwd)"

# If compression is enabled or disabled (e.g. based on the branch main vs sid/add-gzip-compression)
STATE=${1:-enabled}

echo "=========================================="
echo "Building Docker Images"
echo "State: $STATE"
echo "=========================================="
echo ""

# Navigate to repo root
cd "$REPO_ROOT"

# Build each SDK
echo "Building go SDK image..."
docker build -t "go-${STATE}-compression" -f "hack/dev/compression-test/Dockerfile.client-go" . || {
    echo "Error: Failed to build go SDK image"
    exit 1
}
echo "✓ go SDK image built successfully"
echo ""

echo "Building typescript SDK image..."
docker build -t "typescript-${STATE}-compression" -f "hack/dev/compression-test/Dockerfile.client-ts" . || {
    echo "Error: Failed to build typescript SDK image"
    exit 1
}
echo "✓ typescript SDK image built successfully"
echo ""

echo "Building python SDK image..."
docker build -t "python-${STATE}-compression" -f "hack/dev/compression-test/Dockerfile.client-python" . || {
    echo "Error: Failed to build python SDK image"
    exit 1
}
echo "✓ python SDK image built successfully"
echo ""

echo "=========================================="
echo "All images built successfully!"
echo "=========================================="
echo ""
echo "Images created:"
echo "  - go-${STATE}-compression"
echo "  - typescript-${STATE}-compression"
echo "  - python-${STATE}-compression"
