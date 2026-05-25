#!/bin/sh

WORKER_PID=""
FINAL_EXIT=0

cleanup() {
  if [ -n "$WORKER_PID" ]; then
    kill -9 "$WORKER_PID" 2>/dev/null || true
    wait "$WORKER_PID" 2>/dev/null || true
  fi
}
trap cleanup EXIT INT TERM

echo "=== Running integration tests ==="
cd /hatchet/sdks/ruby/src
timeout 300 bundle exec rspec spec/integration/ --format documentation --tag integration
INTEGRATION_EXIT=$?
[ $INTEGRATION_EXIT -ne 0 ] && FINAL_EXIT=$INTEGRATION_EXIT

echo "=== Starting example worker ==="
cd /hatchet/sdks/ruby/examples
HATCHET_CLIENT_WORKER_HEALTHCHECK_ENABLED=true \
HATCHET_CLIENT_WORKER_HEALTHCHECK_PORT=8001 \
bundle exec ruby worker.rb &
WORKER_PID=$!

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

echo "=== Running e2e tests ==="
timeout 1200 bundle exec rspec -f d --fail-fast
E2E_EXIT=$?
[ $E2E_EXIT -ne 0 ] && FINAL_EXIT=$E2E_EXIT

exit $FINAL_EXIT
