#!/bin/bash
# run_all_tests.sh - Run all SDK tests for a given compression state

set -e

STATE=$1
EVENTS_COUNT=${2:-10}  # Default to 10 events if not specified

if [ -z "$STATE" ]; then
    echo "Usage: $0 <state> [events_count]"
    echo "  state: enabled or disabled"
    echo "  events_count: number of events to send (default: 10)"
    exit 1
fi

# Validate required environment variables
if [ -z "$HATCHET_CLIENT_TOKEN" ]; then
    echo "Error: HATCHET_CLIENT_TOKEN environment variable is required"
    echo "Usage: export HATCHET_CLIENT_TOKEN='your-token' && $0 <state>"
    exit 1
fi

# Set default host port for macOS Docker (use IP to avoid IPv6 resolution issues)
if [ -z "$HATCHET_CLIENT_HOST_PORT" ]; then
    # Get the Docker gateway IP (host.docker.internal IPv4)
    GATEWAY_IP=$(docker run --rm alpine getent hosts host.docker.internal 2>/dev/null | awk '{print $1}' | grep -E '^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+' | head -1)
    if [ -z "$GATEWAY_IP" ]; then
        GATEWAY_IP="192.168.65.254"  # Default Docker Desktop gateway
    fi
    export HATCHET_CLIENT_HOST_PORT="${GATEWAY_IP}:7070"
    echo "Using default HATCHET_CLIENT_HOST_PORT: $HATCHET_CLIENT_HOST_PORT"
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "=========================================="
echo "Running all SDK tests (${STATE} compression)"
echo "Events per test: ${EVENTS_COUNT}"
echo "=========================================="
echo ""

# Run tests sequentially in order: python, typescript, go
for SDK in python typescript go; do
    echo "Running ${SDK} SDK test..."
    "$SCRIPT_DIR/run_test.sh" "$SDK" "$STATE" "$EVENTS_COUNT"
    echo ""
    sleep 5  # Brief pause between tests
done

echo "=========================================="
echo "All tests complete for ${STATE} compression"
echo "=========================================="
