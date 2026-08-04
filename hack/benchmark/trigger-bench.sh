#!/usr/bin/env bash
#
# trigger-bench.sh — compare task-trigger write latency between two git refs.
#
# For each ref, this script:
#   1. checks the ref out into a temporary git worktree,
#   2. starts a dedicated postgres container on a random localhost port,
#   3. runs an embedded-mode benchmark driver (built against that worktree's code)
#      which measures per-trigger latency (RunNoWait) under concurrency,
#   4. tears the container down,
# then prints a side-by-side comparison.
#
# Usage:
#   hack/benchmark/trigger-bench.sh                       # origin/main vs HEAD
#   BENCH_TRIGGERS=1000 BENCH_CONCURRENCY=8 hack/benchmark/trigger-bench.sh
#   BENCH_REFS="origin/main mybranch" hack/benchmark/trigger-bench.sh
#   BENCH_PAYLOAD_BYTES=102400 hack/benchmark/trigger-bench.sh   # 100KiB inputs
#
# Requirements: docker, go, python3. Uncommitted changes are NOT benchmarked —
# each ref is benchmarked at its committed state.

set -euo pipefail

BENCH_REFS=${BENCH_REFS:-"origin/main HEAD"}
BENCH_TRIGGERS=${BENCH_TRIGGERS:-300}
BENCH_CONCURRENCY=${BENCH_CONCURRENCY:-4}
BENCH_WARMUP=${BENCH_WARMUP:-50}
BENCH_PAYLOAD_BYTES=${BENCH_PAYLOAD_BYTES:-1024}
POSTGRES_IMAGE=${POSTGRES_IMAGE:-postgres:15.6}

REPO_ROOT=$(git rev-parse --show-toplevel)
WORKDIR=$(mktemp -d /tmp/hatchet-trigger-bench.XXXXXX)

CONTAINERS=()
WORKTREES=()

cleanup() {
    for c in "${CONTAINERS[@]:-}"; do
        docker rm -f "$c" >/dev/null 2>&1 || true
    done
    for w in "${WORKTREES[@]:-}"; do
        git -C "$REPO_ROOT" worktree remove --force "$w" >/dev/null 2>&1 || true
    done
    rm -rf "$WORKDIR"
}
trap cleanup EXIT

free_port() {
    python3 - <<'EOF'
import socket
s = socket.socket()
s.bind(("127.0.0.1", 0))
print(s.getsockname()[1])
s.close()
EOF
}

write_driver() {
    local dir=$1
    cat > "$dir/main.go" <<'EOF'
// Benchmark driver: measures per-trigger (RunNoWait) latency against an embedded
// hatchet engine. Configuration via env; the JSON result is written to RESULT_PATH
// (engine logs go to stdout/stderr, so the result rides a file).
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"

	_ "github.com/hatchet-dev/hatchet/embed"
)

type input struct {
	Payload string `json:"payload"`
}

type output struct {
	OK bool `json:"ok"`
}

func envInt(name string, def int) int {
	if v := os.Getenv(name); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			log.Fatalf("invalid %s: %v", name, err)
		}
		return n
	}
	return def
}

type result struct {
	Triggers          int     `json:"triggers"`
	Concurrency       int     `json:"concurrency"`
	PayloadBytes      int     `json:"payload_bytes"`
	TriggerMeanMs     float64 `json:"trigger_mean_ms"`
	TriggerP50Ms      float64 `json:"trigger_p50_ms"`
	TriggerP90Ms      float64 `json:"trigger_p90_ms"`
	TriggerP99Ms      float64 `json:"trigger_p99_ms"`
	TriggerMaxMs      float64 `json:"trigger_max_ms"`
	TriggersPerSecond float64 `json:"triggers_per_second"`
	CompleteAllSecs   float64 `json:"complete_all_seconds"`
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx := context.Background()

	databaseURL := os.Getenv("DATABASE_URL")
	resultPath := os.Getenv("RESULT_PATH")
	if databaseURL == "" || resultPath == "" {
		return fmt.Errorf("DATABASE_URL and RESULT_PATH are required")
	}

	triggers := envInt("BENCH_TRIGGERS", 300)
	concurrency := envInt("BENCH_CONCURRENCY", 4)
	warmup := envInt("BENCH_WARMUP", 50)
	payloadBytes := envInt("BENCH_PAYLOAD_BYTES", 1024)

	client, err := hatchet.NewClient(hatchet.WithEmbeddedPostgres(databaseURL))
	if err != nil {
		return err
	}
	defer func() { _ = client.Close(context.Background()) }()

	var completed atomic.Int64

	task := client.NewStandaloneTask("trigger-bench", func(ctx hatchet.Context, in input) (output, error) {
		completed.Add(1)
		return output{OK: true}, nil
	})

	worker, err := client.NewWorker("trigger-bench-worker", hatchet.WithWorkflows(task), hatchet.WithSlots(100))
	if err != nil {
		return err
	}

	cleanup, err := worker.Start()
	if err != nil {
		return err
	}
	defer func() { _ = cleanup() }()

	time.Sleep(3 * time.Second)

	in := input{Payload: strings.Repeat("x", payloadBytes)}

	// warmup: not measured, primes caches/pools and the first-partition paths
	for i := 0; i < warmup; i++ {
		if _, err := task.RunNoWait(ctx, in); err != nil {
			return fmt.Errorf("warmup trigger failed: %w", err)
		}
	}

	// measured triggers
	latencies := make([]time.Duration, triggers)
	next := atomic.Int64{}
	wg := sync.WaitGroup{}
	errCh := make(chan error, concurrency)

	start := time.Now()

	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				i := int(next.Add(1)) - 1
				if i >= triggers {
					return
				}

				t0 := time.Now()
				if _, err := task.RunNoWait(ctx, in); err != nil {
					errCh <- fmt.Errorf("trigger %d failed: %w", i, err)
					return
				}
				latencies[i] = time.Since(t0)
			}
		}()
	}

	wg.Wait()
	triggerWall := time.Since(start)

	select {
	case err := <-errCh:
		return err
	default:
	}

	// wait for every run (warmup included) to complete end to end
	total := int64(warmup + triggers)
	deadline := time.Now().Add(5 * time.Minute)
	for completed.Load() < total {
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for completions: %d/%d", completed.Load(), total)
		}
		time.Sleep(100 * time.Millisecond)
	}
	completeWall := time.Since(start)

	sorted := append([]time.Duration{}, latencies...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	pct := func(p float64) float64 {
		idx := int(p * float64(len(sorted)-1))
		return float64(sorted[idx].Microseconds()) / 1000.0
	}

	var sum time.Duration
	for _, l := range sorted {
		sum += l
	}

	res := result{
		Triggers:          triggers,
		Concurrency:       concurrency,
		PayloadBytes:      payloadBytes,
		TriggerMeanMs:     float64(sum.Microseconds()) / float64(len(sorted)) / 1000.0,
		TriggerP50Ms:      pct(0.50),
		TriggerP90Ms:      pct(0.90),
		TriggerP99Ms:      pct(0.99),
		TriggerMaxMs:      pct(1.0),
		TriggersPerSecond: float64(triggers) / triggerWall.Seconds(),
		CompleteAllSecs:   completeWall.Seconds(),
	}

	data, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(resultPath, data, 0o644)
}
EOF
}

bench_ref() {
    local ref=$1
    local name=$2
    local worktree="$WORKDIR/src-$name"
    local driver="$WORKDIR/driver-$name"
    local sha

    sha=$(git -C "$REPO_ROOT" rev-parse --short "$ref")
    echo "==> [$name] $ref @ $sha"

    git -C "$REPO_ROOT" worktree add --detach "$worktree" "$ref" >/dev/null
    WORKTREES+=("$worktree")

    mkdir -p "$driver"
    write_driver "$driver"

    # borrow the embedded example's module graph, pointed at this worktree
    cp "$worktree/sdks/go/examples/embedded/go.mod" "$driver/go.mod"
    cp "$worktree/sdks/go/examples/embedded/go.sum" "$driver/go.sum"
    (
        cd "$driver"
        GOWORK=off go mod edit -module "hatchet-trigger-bench" \
            -dropreplace "github.com/hatchet-dev/hatchet" \
            -replace "github.com/hatchet-dev/hatchet=$worktree"
        GOWORK=off go mod tidy >/dev/null 2>&1
    )

    local port container
    port=$(free_port)
    container="hatchet-trigger-bench-$name-$$"
    docker run -d --name "$container" \
        -e POSTGRES_USER=hatchet -e POSTGRES_PASSWORD=hatchet -e POSTGRES_DB=hatchet \
        -p "127.0.0.1:${port}:5432" "$POSTGRES_IMAGE" >/dev/null
    CONTAINERS+=("$container")

    until docker exec "$container" pg_isready -U hatchet >/dev/null 2>&1; do
        sleep 0.5
    done

    echo "==> [$name] running benchmark (${BENCH_TRIGGERS} triggers, concurrency ${BENCH_CONCURRENCY}, ${BENCH_PAYLOAD_BYTES}B payloads; first run includes migrations)"
    (
        cd "$driver"
        DATABASE_URL="postgresql://hatchet:hatchet@127.0.0.1:${port}/hatchet?sslmode=disable" \
        RESULT_PATH="$WORKDIR/result-$name.json" \
        BENCH_TRIGGERS="$BENCH_TRIGGERS" \
        BENCH_CONCURRENCY="$BENCH_CONCURRENCY" \
        BENCH_WARMUP="$BENCH_WARMUP" \
        BENCH_PAYLOAD_BYTES="$BENCH_PAYLOAD_BYTES" \
        GOWORK=off go run . > "$WORKDIR/log-$name.txt" 2>&1
    ) || {
        echo "!! [$name] benchmark failed; last log lines:"
        tail -20 "$WORKDIR/log-$name.txt"
        exit 1
    }

    docker rm -f "$container" >/dev/null

    echo "==> [$name] done"
}

names=()
i=0
for ref in $BENCH_REFS; do
    name="$(echo "$ref" | tr '/' '-')-$i"
    names+=("$name")
    bench_ref "$ref" "$name"
    i=$((i + 1))
done

echo
python3 - "$WORKDIR" "${names[@]}" <<'EOF'
import json
import sys

workdir, names = sys.argv[1], sys.argv[2:]
results = {}
for name in names:
    with open(f"{workdir}/result-{name}.json") as f:
        results[name] = json.load(f)

fields = [
    ("trigger_mean_ms", "trigger mean (ms)"),
    ("trigger_p50_ms", "trigger p50 (ms)"),
    ("trigger_p90_ms", "trigger p90 (ms)"),
    ("trigger_p99_ms", "trigger p99 (ms)"),
    ("trigger_max_ms", "trigger max (ms)"),
    ("triggers_per_second", "triggers/sec"),
    ("complete_all_seconds", "all complete (s)"),
]

base = names[0]
width = max(len(label) for _, label in fields)
header = f"{'':{width}}" + "".join(f"{n:>24}" for n in names)
if len(names) == 2:
    header += f"{'delta':>12}"
print(header)
for key, label in fields:
    row = f"{label:{width}}"
    for n in names:
        row += f"{results[n][key]:>24.2f}"
    if len(names) == 2:
        a, b = results[names[0]][key], results[names[1]][key]
        pct = ((b - a) / a * 100) if a else 0.0
        row += f"{pct:>+11.1f}%"
    print(row)

meta = results[base]
print(f"\n({meta['triggers']} triggers, concurrency {meta['concurrency']}, {meta['payload_bytes']}B payloads)")
EOF
