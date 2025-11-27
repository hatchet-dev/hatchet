#!/bin/bash
# build_and_test.sh - Build images and run tests for current branch

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
REPO_ROOT="$(cd "$TEST_DIR/../../.." && pwd)"

STATE=${1:-enabled}
HATCHET_CLIENT_TOKEN=${HATCHET_CLIENT_TOKEN:-""}

if [ -z "$HATCHET_CLIENT_TOKEN" ]; then
    echo "Error: HATCHET_CLIENT_TOKEN environment variable is required"
    echo "Usage: export HATCHET_CLIENT_TOKEN='your-token' && $0 [enabled|disabled]"
    exit 1
fi

echo "=========================================="
echo "Building and Testing Compression Suite"
echo "State: $STATE"
echo "=========================================="
echo ""

# Navigate to repo root
cd "$REPO_ROOT"

echo "Building Docker images..."
echo ""

# Build Go SDK
echo "Building Go SDK image..."
docker build -t "go-${STATE}-compression" -f hack/dev/compression-test/Dockerfile.client-go . || {
    echo "Error: Failed to build Go SDK image"
    exit 1
}

# Build TypeScript SDK
echo "Building TypeScript SDK image..."
docker build -t "typescript-${STATE}-compression" -f hack/dev/compression-test/Dockerfile.client-ts . || {
    echo "Error: Failed to build TypeScript SDK image"
    exit 1
}

# Build Python SDK
echo "Building Python SDK image..."
docker build -t "python-${STATE}-compression" -f hack/dev/compression-test/Dockerfile.client-python . || {
    echo "Error: Failed to build Python SDK image"
    exit 1
}

echo ""
echo "All images built successfully!"
echo ""

# Navigate to test directory
cd "$TEST_DIR"

# Run setup if needed
if ! docker network ls | grep -q hatchet-test; then
    echo "Running setup..."
    ./scripts/setup.sh
fi

echo ""
echo "Running tests with HATCHET_CLIENT_TOKEN..."
echo ""

# Export token for all test runs
export HATCHET_CLIENT_TOKEN

# Run all tests
./scripts/run_all_tests.sh "$STATE"

echo ""
echo "Generating report..."
./scripts/generate_report.sh

echo ""
echo "=========================================="
echo "Build and test complete!"
echo "=========================================="
