#!/bin/bash
# run_test.sh - Run compression test for a specific SDK

set -e

SDK=$1
STATE=$2
EVENTS_COUNT=${3:-10}  # Default to 10 events if not specified

if [ -z "$SDK" ] || [ -z "$STATE" ]; then
    echo "Usage: $0 <sdk> <state> [events_count]"
    echo "  sdk: go, typescript, or python"
    echo "  state: enabled or disabled"
    echo "  events_count: number of events to send (default: 10)"
    exit 1
fi

if [ "$STATE" != "enabled" ] && [ "$STATE" != "disabled" ]; then
    echo "Error: state must be 'enabled' or 'disabled'"
    exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
RESULTS_DIR="$TEST_DIR/results/$STATE"
CLIENT_CONTAINER="hatchet-client-${SDK}"

mkdir -p "$RESULTS_DIR"

echo "=========================================="
echo "Running ${SDK} SDK test (${STATE} compression)"
echo "Events: ${EVENTS_COUNT}"
echo "=========================================="

# Validate required environment variables
if [ -z "$HATCHET_CLIENT_TOKEN" ]; then
    echo "Error: HATCHET_CLIENT_TOKEN environment variable is required"
    echo "Usage: export HATCHET_CLIENT_TOKEN='your-token' && $0 <sdk> <state>"
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

# Check if client image exists
IMAGE_NAME="${SDK}-${STATE}-compression"
if ! docker image inspect "$IMAGE_NAME" >/dev/null 2>&1; then
    echo "Error: Docker image '$IMAGE_NAME' not found"
    echo "Please build it first. See README.md for build instructions."
    exit 1
fi

# Clean up any existing container
docker rm -f "$CLIENT_CONTAINER" 2>/dev/null || true

# Start the client container in background
echo "Starting ${SDK} client container..."
cd "$TEST_DIR"

# Export environment variables for docker-compose
export COMPRESSION_STATE="$STATE"
export HATCHET_CLIENT_TOKEN="${HATCHET_CLIENT_TOKEN}"
export HATCHET_CLIENT_HOST_PORT="${HATCHET_CLIENT_HOST_PORT}"
export HATCHET_CLIENT_SERVER_URL="${HATCHET_CLIENT_SERVER_URL:-http://localhost:8080}"
export HATCHET_CLIENT_API_URL="${HATCHET_CLIENT_API_URL:-${HATCHET_CLIENT_SERVER_URL:-http://localhost:8080}}"
export HATCHET_CLIENT_NAMESPACE="${HATCHET_CLIENT_NAMESPACE:-compression-test}"
export TEST_EVENTS_COUNT="${EVENTS_COUNT}"

# For Go test: --events is the rate (events per second), not total count
# To get approximately EVENTS_COUNT events, we need: duration = EVENTS_COUNT / rate
# We'll use 10 events per second rate
# Note: Due to timing, this may emit slightly more than EVENTS_COUNT events
EVENTS_PER_SECOND=10
# Calculate duration: events_count / rate (rounded up to nearest second)
DURATION_SECONDS=$((EVENTS_COUNT / EVENTS_PER_SECOND))
# Round up if there's a remainder
if [ $((EVENTS_COUNT % EVENTS_PER_SECOND)) -gt 0 ]; then
    DURATION_SECONDS=$((DURATION_SECONDS + 1))
fi
# Ensure minimum of 1 second
if [ $DURATION_SECONDS -lt 1 ]; then
    DURATION_SECONDS=1
fi
export TEST_EVENTS_RATE="${EVENTS_PER_SECOND}"
export TEST_DURATION="${DURATION_SECONDS}s"
# Wait time should be duration + buffer for events to complete processing
# Add buffer for processing time (events take time to execute)
export TEST_WAIT="$((DURATION_SECONDS + 5))s"

# Run docker-compose with environment variables
docker-compose run -d \
    --name "$CLIENT_CONTAINER" \
    "client-${SDK}" > /dev/null 2>&1

# Wait a moment for container to start
sleep 2

# Calculate monitoring duration based on test duration
# For Go: use TEST_DURATION, for others: calculate from events count
if [ "$SDK" = "go" ]; then
    # Parse duration (e.g., "5s" -> 5)
    MONITOR_DURATION=$(echo "$TEST_DURATION" | sed 's/s$//')
    MONITOR_DURATION=$((MONITOR_DURATION + 10))  # Add buffer
else
    # For Python/TypeScript: events / 10 events per second + buffer
    EVENTS_PER_SECOND=10
    MONITOR_DURATION=$((EVENTS_COUNT / EVENTS_PER_SECOND + 15))  # Add 15 second buffer
fi

# Start network monitoring
echo "Starting network monitoring..."
"$SCRIPT_DIR/monitor_network.sh" "$CLIENT_CONTAINER" "$MONITOR_DURATION" "$RESULTS_DIR/${SDK}_network.log" &
MONITOR_PID=$!

# Stream logs in real-time (limit to last 10 lines to avoid huge files)
echo "Streaming container logs (press Ctrl+C to stop streaming, container will continue)..."
docker logs -f --tail 10 "$CLIENT_CONTAINER" 2>&1 | tee "$RESULTS_DIR/${SDK}_test.log" &
LOGS_PID=$!

# Wait for container to complete (with timeout)
echo "Waiting for test to complete..."
# Increase timeout for TypeScript (it may take longer to process)
if [ "$SDK" = "typescript" ]; then
    TIMEOUT=180  # 3 minutes timeout for TypeScript
else
    TIMEOUT=120  # 2 minutes timeout for others
fi
ELAPSED=0
while [ $ELAPSED -lt $TIMEOUT ]; do
    if ! docker ps --format '{{.Names}}' | grep -q "^${CLIENT_CONTAINER}$"; then
        # Container stopped
        break
    fi
    sleep 1
    ELAPSED=$((ELAPSED + 1))
done

# If container is still running after timeout, kill it
if docker ps --format '{{.Names}}' | grep -q "^${CLIENT_CONTAINER}$"; then
    echo "Warning: Test timed out after ${TIMEOUT}s, stopping container..."
    docker stop "$CLIENT_CONTAINER" > /dev/null 2>&1 || true
    sleep 2  # Give it a moment to stop
fi

# Stop log streaming
kill $LOGS_PID 2>/dev/null || true
wait $LOGS_PID 2>/dev/null || true

# Wait for monitoring to complete - it needs to finish to write the summary file
# The monitoring script runs for MONITOR_DURATION seconds, then writes the summary
# Wait for it to complete (MONITOR_DURATION + small buffer for file I/O)
MONITOR_TIMEOUT=$((MONITOR_DURATION + 5))
ELAPSED=0
while [ $ELAPSED -lt $MONITOR_TIMEOUT ]; do
    if ! kill -0 $MONITOR_PID 2>/dev/null; then
        # Monitoring script finished
        break
    fi
    sleep 1
    ELAPSED=$((ELAPSED + 1))
done

# If still running after timeout, force kill it
if kill -0 $MONITOR_PID 2>/dev/null; then
    kill -KILL $MONITOR_PID 2>/dev/null || true
fi
wait $MONITOR_PID 2>/dev/null || true

# Clean up container
docker rm -f "$CLIENT_CONTAINER" > /dev/null 2>&1 || true

# Helper function to format bytes in human-readable format
format_bytes() {
    local bytes=$1
    if [ $bytes -ge 1099511627776 ]; then
        awk "BEGIN {printf \"%.2f TB\", $bytes / 1099511627776}"
    elif [ $bytes -ge 1073741824 ]; then
        awk "BEGIN {printf \"%.2f GB\", $bytes / 1073741824}"
    elif [ $bytes -ge 1048576 ]; then
        awk "BEGIN {printf \"%.2f MB\", $bytes / 1048576}"
    elif [ $bytes -ge 1024 ]; then
        awk "BEGIN {printf \"%.2f KB\", $bytes / 1024}"
    else
        echo "${bytes} B"
    fi
}

# Extract network summary
if [ -f "$RESULTS_DIR/${SDK}_network.log.summary" ]; then
    source "$RESULTS_DIR/${SDK}_network.log.summary"
    RX_FORMATTED=$(format_bytes $RX_BYTES)
    TX_FORMATTED=$(format_bytes $TX_BYTES)
    TOTAL_FORMATTED=$(format_bytes $TOTAL_BYTES)
    echo ""
    echo "=== Test Results ==="
    echo "SDK: $SDK"
    echo "State: $STATE"
    echo "RX Bytes: $RX_FORMATTED ($RX_BYTES bytes)"
    echo "TX Bytes: $TX_FORMATTED ($TX_BYTES bytes)"
    echo "Total Bytes: $TOTAL_FORMATTED ($TOTAL_BYTES bytes)"
    echo ""
    echo "Results saved to: $RESULTS_DIR/${SDK}_network.log.summary"
else
    echo "Warning: Could not find network summary file"
fi

echo "Test complete for ${SDK} SDK (${STATE})"
