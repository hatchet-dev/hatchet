#!/usr/bin/env bash
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

DATABASE_URL="${DATABASE_URL:-postgres://hatchet:hatchet@127.0.0.1:5431/hatchet?sslmode=disable}"
RUNS="${RUNS:-30}"
export DATABASE_URL

BIN="$(mktemp -d)"
RUNDIR="/tmp/hatchet-embed-fleet"
rm -rf "$RUNDIR"; mkdir -p "$RUNDIR"
pids=()

cleanup() {
  echo "shutting down..."
  for pid in "${pids[@]:-}"; do kill "$pid" 2>/dev/null || true; done
  wait 2>/dev/null || true
  rm -rf "$BIN"
}
trap cleanup EXIT INT TERM

echo "building engine + worker + trigger..."
(cd embed/example && go build -o "$BIN/engine" ./engine)
(cd embed/example && go build -o "$BIN/worker" ./worker)
(cd embed/example && go build -o "$BIN/trigger" ./trigger)

start_engine() {
  local idx="$1" migrate="$2"
  local api=$((8090 + idx)) grpc=$((7070 + idx))
  local out="$RUNDIR/engine-$idx.json"
  rm -f "$out"
  echo "starting engine $idx (api=$api grpc=$grpc migrate=$migrate)"
  API_PORT="$api" GRPC_PORT="$grpc" OUTPUT_FILE="$out" RUN_MIGRATIONS="$migrate" \
    "$BIN/engine" &
  pids+=($!)
  for _ in $(seq 1 60); do [[ -s "$out" ]] && return 0; sleep 1; done
  echo "engine $idx did not come up"; exit 1
}

start_worker() {
  local idx="$1"
  echo "starting worker $idx -> engine $idx"
  WORKER_NAME="worker-$idx" ENGINE_FILE="$RUNDIR/engine-$idx.json" "$BIN/worker" &
  pids+=($!)
}

start_engine 0 true
sleep 5
start_engine 1 false
sleep 5
start_engine 2 false

start_worker 0
start_worker 1
start_worker 2

echo "waiting for workers to register..."
sleep 6

RUNS="$RUNS" ENGINE_FILE="$RUNDIR/engine-0.json" "$BIN/trigger"

TENANT=$(python3 -c "import json;print(json.load(open('$RUNDIR/engine-0.json'))['tenantID'])")
TOKEN=$(python3 -c "import json;print(json.load(open('$RUNDIR/engine-0.json'))['token'])")
API=$(python3 -c "import json;print(json.load(open('$RUNDIR/engine-0.json'))['apiURL'])")
echo
echo "REST API is live on each engine (no frontend). Recent runs from engine 0:"
curl -s -H "Authorization: Bearer $TOKEN" "$API/api/v1/stable/tenants/$TENANT/workflow-runs?only_tasks=true&limit=5&since=2020-01-01T00:00:00Z" \
  | python3 -c "import sys,json; d=json.load(sys.stdin); print(f'  API returned {len(d.get(\"rows\",[]))} run(s) on page 1 of {d.get(\"pagination\",{}).get(\"num_pages\",\"?\")}')" 2>/dev/null \
  || echo "  (API responded)"

echo
echo "cluster still running. re-trigger anytime with:"
echo "  RUNS=50 ENGINE_FILE=$RUNDIR/engine-0.json go run -C embed/example ./trigger"
echo "Ctrl+C to stop."
wait
