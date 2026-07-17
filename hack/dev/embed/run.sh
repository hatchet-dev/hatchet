#!/usr/bin/env bash
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

DATABASE_URL="${DATABASE_URL:-postgres://hatchet:hatchet@127.0.0.1:5431/hatchet?sslmode=disable}"
RUNS="${RUNS:-30}"
WORKERS="${WORKERS:-3}"
export DATABASE_URL

BIN="$(mktemp -d)"
pids=()

cleanup() {
  echo "shutting down..."
  for pid in "${pids[@]:-}"; do kill "$pid" 2>/dev/null || true; done
  wait 2>/dev/null || true
  rm -rf "$BIN"
}
trap cleanup EXIT INT TERM

echo "building worker + trigger..."
(cd embed/example && go build -o "$BIN/worker" ./worker)
(cd embed/example && go build -o "$BIN/trigger" ./trigger)

for idx in $(seq 0 $((WORKERS - 1))); do
  echo "starting worker $idx (embedded engine, auto-selected ports)"
  WORKER_NAME="worker-$idx" "$BIN/worker" &
  pids+=($!)
  sleep 3
done

echo "waiting for the fleet to settle..."
sleep 6

RUNS="$RUNS" "$BIN/trigger"

echo
echo "every process embeds its own engine; Postgres is the only shared coordination layer."
echo "re-trigger anytime with:"
echo "  RUNS=50 go run -C embed/example ./trigger"
echo "Ctrl+C to stop."
wait
