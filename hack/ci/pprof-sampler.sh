#!/bin/sh
# Pulls CPU profiles and goroutine dumps from a Go process's pprof endpoint
# while a job step runs, so a slow run can be read from inside the process:
# which functions burned the CPU it did get, and how many goroutines were
# waiting when it did not.
#
# Usage:
#   pprof-sampler.sh start <dir> <addr>   sample in the background, pid in <dir>/sampler.pid
#   pprof-sampler.sh stop <dir>           stop sampling
#   pprof-sampler.sh summarize <dir>      merge the CPU profiles and print the top functions
#
# addr is host:port of the pprof endpoint. Each CPU profile spans
# PPROF_SAMPLE_SECONDS (30) and a goroutine dump is taken alongside it.
# Needs curl and, for summarize, the Go toolchain.

set -u

seconds="${PPROF_SAMPLE_SECONDS:-30}"

sample_loop() {
  dir="$1"
  addr="$2"

  # the endpoint comes up once the test binary starts, after the build
  until curl -sf -o /dev/null "http://$addr/debug/pprof/"; do
    sleep 2
  done
  echo "# seconds: $seconds" >> "$dir/sampler.log"
  echo "$(date -u +%H:%M:%S) pprof endpoint up" >> "$dir/sampler.log"

  while :; do
    stamp="$(date -u +%H%M%S)"
    curl -sf -o "$dir/goroutines-$stamp.txt" "http://$addr/debug/pprof/goroutine?debug=1" || true
    # written to a temporary name and renamed when complete, so a profile cut
    # short by stop never reaches the merge
    if curl -sf -o "$dir/cpu-$stamp.tmp" "http://$addr/debug/pprof/profile?seconds=$seconds"; then
      mv "$dir/cpu-$stamp.tmp" "$dir/cpu-$stamp.pb.gz"
    else
      rm -f "$dir/cpu-$stamp.tmp"
      # the process is gone once the tests end
      if ! curl -sf -o /dev/null "http://$addr/debug/pprof/"; then
        echo "$(date -u +%H:%M:%S) pprof endpoint gone" >> "$dir/sampler.log"
        exit 0
      fi
      sleep 2
    fi
  done
}

case "${1:-}" in
  start)
    dir="$2"
    addr="$3"
    mkdir -p "$dir"
    sample_loop "$dir" "$addr" > /dev/null 2>&1 &
    echo $! > "$dir/sampler.pid"
    echo "pprof sampler started (pid $!, ${seconds}s profiles from $addr) -> $dir"
    ;;
  stop)
    dir="$2"
    if [ -f "$dir/sampler.pid" ]; then
      pid="$(cat "$dir/sampler.pid")"
      # the loop's in-flight curl is a child of the loop; stop both
      pkill -P "$pid" 2>/dev/null || true
      kill "$pid" 2>/dev/null || true
      rm -f "$dir/sampler.pid" "$dir"/cpu-*.tmp
    fi
    ;;
  summarize)
    dir="$2"
    count="$(ls "$dir"/cpu-*.pb.gz 2>/dev/null | wc -l | tr -d ' ')"
    if [ "$count" = "0" ]; then
      echo "no cpu profiles captured"
      exit 0
    fi
    each="$(awk '/^# seconds:/ { print $3 }' "$dir/sampler.log" 2>/dev/null)"
    echo "cpu profiles: $count of ${each:-$seconds}s each, merged"
    go tool pprof -proto "$dir"/cpu-*.pb.gz > "$dir/cpu-merged.pb.gz" 2>/dev/null
    echo
    echo "top functions by own time, scheduler idle excluded:"
    go tool pprof -top -nodecount=25 -ignore='runtime\.findRunnable' "$dir/cpu-merged.pb.gz" 2>/dev/null | sed -n '4,32p'
    echo
    echo "top hatchet functions by cumulative time (flat, flat%, cum, cum%):"
    go tool pprof -top -cum -nodecount=500 "$dir/cpu-merged.pb.gz" 2>/dev/null | awk '/hatchet-dev\/hatchet/ { printf "  %8s %6s %8s %6s  %s\n", $1, $2, $4, $5, $6 }' | head -25
    echo
    echo "goroutines per dump:"
    for f in "$dir"/goroutines-*.txt; do
      [ -f "$f" ] || continue
      total="$(awk '/^goroutine profile: total/ { print $4 }' "$f")"
      printf "  %s: %s\n" "$(basename "$f" .txt | sed 's/goroutines-//')" "${total:-?}"
    done
    ;;
  *)
    echo "usage: pprof-sampler.sh start <dir> <addr> | stop <dir> | summarize <dir>" >&2
    exit 2
    ;;
esac
