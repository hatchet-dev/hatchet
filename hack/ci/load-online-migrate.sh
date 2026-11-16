#!/usr/bin/env bash

# Drives the load-online-migrate CI scenario in the order a real deploy takes:
# migrations first, engines second.
#
#   1. start the BASE commit's engine on the base schema and put it under load
#   2. after the warm-up window, apply this branch's migrations (up, down to
#      the base version, up again) while the base engine keeps serving
#   3. keep the load running on base engine + new schema until the load test
#      checks that every pushed event executed exactly once
#   4. stop the base engine, start this branch's engine on the migrated
#      schema and run the same load test again
#
# Each load test phase fails on lost or duplicated task runs (the load test
# counts distinct (task, event) executions), so a new binary that needs a
# schema object the migration has not created yet (the reverse of production)
# can no longer be what this job tests, and a migration the running engine
# cannot live with is caught in phase 3.
#
# Required environment:
#   BASE_BIN                directory holding the base commit's hatchet-engine
#   PR_BIN                  directory holding this checkout's hatchet-engine,
#                           hatchet-migrate and hatchet-loadtest
#   BASE_MIGRATION_VERSION  latest goose version on the base commit
#   HATCHET_CLIENT_TOKEN    API token for the load test
#   DATABASE_URL and the SERVER_* variables the engine and migrator read
#
# Optional:
#   CONFIG_DIR    engine config directory (default ./generated)
#   LOG_DIR       where engine and load test logs go (default /tmp)
#   GRPC_PORT     engine gRPC port to wait on (default $SERVER_GRPC_PORT or 7077)
#   LOAD_EVENTS   events per second (default 10)
#   LOAD_DURATION emit window per phase (default 240s)
#   LOAD_WAIT     time to wait for stragglers after emitting (default 60s)
#   WARMUP_SLEEP  seconds of load on the base schema before migrating (default 30)

set -euo pipefail

: "${BASE_BIN:?BASE_BIN must point at the binaries built from the base commit}"
: "${PR_BIN:?PR_BIN must point at the binaries built from this checkout}"
: "${BASE_MIGRATION_VERSION:?BASE_MIGRATION_VERSION must be the latest goose version on the base commit}"
: "${HATCHET_CLIENT_TOKEN:?HATCHET_CLIENT_TOKEN must be set}"

CONFIG_DIR="${CONFIG_DIR:-./generated}"
LOG_DIR="${LOG_DIR:-/tmp}"
GRPC_PORT="${GRPC_PORT:-${SERVER_GRPC_PORT:-7077}}"
LOAD_EVENTS="${LOAD_EVENTS:-10}"
LOAD_DURATION="${LOAD_DURATION:-240s}"
LOAD_WAIT="${LOAD_WAIT:-60s}"
# Keep this above the load test's 20s registration timeout so PutWorkflow has
# to succeed before the branch migrations are applied.
WARMUP_SLEEP="${WARMUP_SLEEP:-30}"

mkdir -p "$LOG_DIR"

ENGINE_PID=""
LOADTEST_PID=""
WATCHER_PID=""

dump_logs() {
  local phase="$1"
  echo "=== Load test logs ($phase) ==="
  cat "$LOG_DIR/loadtest-$phase.log"
  echo "=== Engine logs ($phase) ==="
  cat "$LOG_DIR/engine-$phase.log"
}

# The pid files are for the workflow's teardown step, which has to kill the
# engine and the load test when this script is aborted by the job timeout.
start_engine() {
  local phase="$1" binary="$2"

  echo "Starting $phase engine ($binary)..."
  "$binary" --config "$CONFIG_DIR" > "$LOG_DIR/engine-$phase.log" 2>&1 &
  ENGINE_PID=$!
  echo "$ENGINE_PID" > "$LOG_DIR/engine.pid"

  echo "Waiting for $phase engine gRPC port $GRPC_PORT..."
  for _ in $(seq 1 60); do
    if ! kill -0 "$ENGINE_PID" 2>/dev/null; then
      echo "$phase engine exited before becoming ready:"
      tail -50 "$LOG_DIR/engine-$phase.log"
      exit 1
    fi

    if nc -z localhost "$GRPC_PORT"; then
      echo "$phase engine is accepting connections on port $GRPC_PORT"
      return 0
    fi

    sleep 1
  done

  echo "Timed out waiting for $phase engine gRPC port $GRPC_PORT"
  tail -50 "$LOG_DIR/engine-$phase.log"
  exit 1
}

# The next engine binds the same port, so stopping is not done until the
# port is released, not merely when the process is gone.
stop_engine() {
  local phase="$1"

  echo "Stopping $phase engine (pid $ENGINE_PID)..."
  kill "$ENGINE_PID" 2>/dev/null || true
  for _ in $(seq 1 60); do
    if ! kill -0 "$ENGINE_PID" 2>/dev/null; then
      break
    fi
    sleep 1
  done
  if kill -0 "$ENGINE_PID" 2>/dev/null; then
    echo "$phase engine did not exit within 60s, killing it"
    kill -9 "$ENGINE_PID" 2>/dev/null || true
  fi
  wait "$ENGINE_PID" 2>/dev/null || true

  for _ in $(seq 1 30); do
    if ! nc -z localhost "$GRPC_PORT"; then
      echo "$phase engine stopped"
      rm -f "$LOG_DIR/engine.pid"
      return 0
    fi
    sleep 1
  done

  echo "port $GRPC_PORT is still bound after stopping the $phase engine"
  exit 1
}

# Without the watcher an engine crash only surfaces minutes later, at the end
# of the window, as lost events; the marker file lets wait_load tell the two
# apart.
start_load() {
  local phase="$1"

  echo "Starting load test ($phase): $LOAD_EVENTS events/s for $LOAD_DURATION, then waiting $LOAD_WAIT"
  HATCHET_CLIENT_TOKEN="$HATCHET_CLIENT_TOKEN" \
  HATCHET_CLIENT_TLS_STRATEGY=none \
  HATCHET_CLIENT_HOST_PORT="localhost:$GRPC_PORT" \
  "$PR_BIN/hatchet-loadtest" loadtest -e "$LOAD_EVENTS" -d "$LOAD_DURATION" -w "$LOAD_WAIT" -s 100 \
    --registrationTimeout 20s > "$LOG_DIR/loadtest-$phase.log" 2>&1 &
  LOADTEST_PID=$!
  echo "$LOADTEST_PID" > "$LOG_DIR/loadtest.pid"

  (
    while kill -0 "$LOADTEST_PID" 2>/dev/null; do
      if ! kill -0 "$ENGINE_PID" 2>/dev/null; then
        echo "$phase engine died during load test:"
        tail -50 "$LOG_DIR/engine-$phase.log"
        touch "$LOG_DIR/engine-died-$phase"
        kill "$LOADTEST_PID" 2>/dev/null || true
        exit 1
      fi
      sleep 5
    done
  ) &
  WATCHER_PID=$!
}

wait_load() {
  local phase="$1" rc

  echo "Waiting for load test ($phase) to complete..."
  set +e
  wait "$LOADTEST_PID"
  rc=$?
  set -e

  kill "$WATCHER_PID" 2>/dev/null || true
  wait "$WATCHER_PID" 2>/dev/null || true
  rm -f "$LOG_DIR/loadtest.pid"

  if [ -f "$LOG_DIR/engine-died-$phase" ]; then
    echo "Load test ($phase) aborted because the engine died"
    dump_logs "$phase"
    exit 1
  fi

  if [ "$rc" -ne 0 ]; then
    echo "Load test ($phase) failed with exit code $rc"
    dump_logs "$phase"
    exit "$rc"
  fi

  echo "Load test ($phase) passed"
}

echo "=== Phase 1: base engine on the base schema (version $BASE_MIGRATION_VERSION) ==="
start_engine base "$BASE_BIN/hatchet-engine"
start_load base

echo "Waiting ${WARMUP_SLEEP}s for the load test to get started on the base schema..."
sleep "$WARMUP_SLEEP"

echo "=== Phase 2: apply branch migrations while the base engine serves ==="
echo "Applying branch migrations..."
"$PR_BIN/hatchet-migrate"
echo "Branch migrations applied"

# A migration is rolled back before the new engines roll out, so the down
# path has to work with the base engine live as well.
echo "Migrating down to base version $BASE_MIGRATION_VERSION..."
"$PR_BIN/hatchet-migrate" --down "$BASE_MIGRATION_VERSION"
echo "Down migration successful"

echo "Re-applying branch migrations..."
"$PR_BIN/hatchet-migrate"
echo "Re-migration successful"

echo "=== Phase 3: base engine on the migrated schema ==="
wait_load base
stop_engine base

echo "=== Phase 4: branch engine on the migrated schema ==="
start_engine pr "$PR_BIN/hatchet-engine"
start_load pr
wait_load pr
stop_engine pr

echo "Load test passed in every phase"
