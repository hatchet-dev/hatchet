#!/bin/sh
set -e

# Run integration tests (no worker needed)
echo "=== Running integration tests ==="
cd /hatchet/sdks/ruby/src
bundle exec rspec spec/integration/ --format documentation --tag integration

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
        kill "$WORKER_PID" 2>/dev/null || true
        exit 1
    fi
    sleep 1
done

# Run e2e tests
echo "=== Running e2e tests ==="
bundle exec rspec -f d --fail-fast
E2E_STATUS=$?

kill "$WORKER_PID" 2>/dev/null || true
exit $E2E_STATUS
