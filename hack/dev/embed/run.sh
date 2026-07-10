#!/usr/bin/env bash
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

DATABASE_URL="${DATABASE_URL:-postgres://hatchet:hatchet@127.0.0.1:5431/hatchet?sslmode=disable}"
KEYSET_DIR="${KEYSET_DIR:-hack/dev/encryption-keys}"
HATCHET_EMBED_VERSION="${HATCHET_EMBED_VERSION:-v0.83.4}"
RUNS="${RUNS:-30}"
export DATABASE_URL KEYSET_DIR HATCHET_EMBED_VERSION

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

if [[ ! -f "$KEYSET_DIR/master.key" ]]; then
  echo "generating shared keysets in $KEYSET_DIR"
  mkdir -p "$KEYSET_DIR"
  go run ./cmd/hatchet-admin keyset create-local-keys --key-dir "$KEYSET_DIR"
fi

echo "building engine + worker + trigger..."
go build -o "$BIN/engine" ./embed/example/engine
go build -o "$BIN/worker" ./embed/example/worker
go build -o "$BIN/trigger" ./embed/example/trigger

start_engine() {
  local idx="$1" migrate="$2"
  local api=$((8080 + idx)) grpc=$((7070 + idx))
  local out="$RUNDIR/engine-$idx.json"
  rm -f "$out"
  echo "starting engine $idx (api=$api grpc=$grpc migrate=$migrate)"
  HATCHET_KEYSET_DIR="$KEYSET_DIR" API_PORT="$api" GRPC_PORT="$grpc" \
    OUTPUT_FILE="$out" RUN_MIGRATIONS="$migrate" \
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

TENANT=707d0855-80ab-4e1f-a156-f1c4546cbf52
TOKEN=$(python3 -c "import json;print(json.load(open('$RUNDIR/engine-0.json'))['token'])")
echo
echo "REST API is live on each engine (no frontend). Recent runs from engine 0:"
curl -s -H "Authorization: Bearer $TOKEN" "http://localhost:8080/api/v1/stable/tenants/$TENANT/workflow-runs?only_tasks=true&limit=5&since=2020-01-01T00:00:00Z" \
  | python3 -c "import sys,json; d=json.load(sys.stdin); print(f'  API returned {len(d.get(\"rows\",[]))} run(s) on page 1 of {d.get(\"pagination\",{}).get(\"num_pages\",\"?\")}')" 2>/dev/null \
  || echo "  (API responded)"

echo
echo "cluster still running. re-trigger anytime with:"
echo "  RUNS=50 ENGINE_FILE=$RUNDIR/engine-0.json go run ./embed/example/trigger"
echo "Ctrl+C to stop."
wait
