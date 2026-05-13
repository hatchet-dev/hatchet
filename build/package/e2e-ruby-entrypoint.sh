#!/bin/sh
set -e

export HATCHET_CLIENT_NAMESPACE=ruby

WORKER_PID=""

cleanup() {
  if [ -n "$WORKER_PID" ]; then
    kill -9 "$WORKER_PID" 2>/dev/null || true
    wait "$WORKER_PID" 2>/dev/null || true
  fi
}
trap cleanup EXIT INT TERM

# Run integration tests (no worker needed)
echo "=== Running integration tests ==="
cd /hatchet/sdks/ruby/src
timeout 300 bundle exec rspec spec/integration/ --format documentation --tag integration

# Start the example worker in the background
echo "=== Starting example worker ==="
cd /hatchet/sdks/ruby/examples
HATCHET_CLIENT_WORKER_HEALTHCHECK_ENABLED=true \
HATCHET_CLIENT_WORKER_HEALTHCHECK_PORT=8001 \
bundle exec ruby worker.rb &
WORKER_PID=$!

# Wait for worker health
for i in $(seq 1 60); do
    if curl -sf http://localhost:8001/health > /dev/null 2>&1; then
        echo "Worker healthy after ${i}s"
        break
    fi
    if [ "$i" -eq 60 ]; then
        echo "Worker failed to start within 60s"
        exit 1
    fi
    sleep 1
done

# Run e2e tests
echo "=== Running e2e tests ==="
timeout 900 bundle exec rspec -f d --fail-fast
