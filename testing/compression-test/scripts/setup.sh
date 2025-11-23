#!/bin/bash
# setup.sh - Initial setup script

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "Setting up compression test environment..."

# Create results directories
mkdir -p "$TEST_DIR/results/baseline"
mkdir -p "$TEST_DIR/results/enabled"

# Note: Using host network mode, so no network creation needed

# Check for required tools
echo ""
echo "Checking for required tools..."

MISSING_TOOLS=()

if ! command -v docker >/dev/null 2>&1; then
    MISSING_TOOLS+=("docker")
fi

if ! command -v docker-compose >/dev/null 2>&1 && ! docker compose version >/dev/null 2>&1; then
    MISSING_TOOLS+=("docker-compose")
fi

if [ ${#MISSING_TOOLS[@]} -gt 0 ]; then
    echo "Error: Missing required tools: ${MISSING_TOOLS[*]}"
    exit 1
fi

echo "✓ All required tools are installed"

echo ""
echo "Setup complete!"
echo ""
echo "Next steps:"
echo "1. Build baseline images (main branch):"
echo "   git checkout main"
echo "   docker build -t go-disabled-compression -f Dockerfile.client-go .."
echo "   docker build -t typescript-disabled-compression -f Dockerfile.client-ts .."
echo "   docker build -t python-disabled-compression -f Dockerfile.client-python .."
echo ""
echo "2. Build compression images (current branch):"
echo "   git checkout <your-compression-branch>"
echo "   docker build -t go-enabled-compression -f Dockerfile.client-go .."
echo "   docker build -t typescript-enabled-compression -f Dockerfile.client-ts .."
echo "   docker build -t python-enabled-compression -f Dockerfile.client-python .."
echo ""
echo "3. Start engine and run tests:"
echo "   ./scripts/run_all_tests.sh disabled"
echo "   ./scripts/run_all_tests.sh enabled"
echo "   ./scripts/generate_report.sh"
