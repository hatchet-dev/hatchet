#!/bin/sh
# Snapshots pg_stat_statements and pg_stat_database from the load test's
# Postgres container while it runs, so a slow run can be read against where
# the database spent its time. The container is created and destroyed by the
# test binary, so the samples are taken through docker exec while it is alive
# and the last snapshot is what the summary reports.
#
# Usage:
#   pgstat-sampler.sh start <dir> [image]   sample in the background, pid in <dir>/sampler.pid
#   pgstat-sampler.sh stop <dir>            stop sampling
#   pgstat-sampler.sh summarize <dir>       print the last snapshot's totals and top statements
#
# image is the ancestor image of the container to sample (postgres:17-alpine
# by default). Samples every PGSTAT_SAMPLE_INTERVAL seconds (30). The harness
# starts Postgres with pg_stat_statements preloaded; the extension is created
# here on first contact.

set -u

interval="${PGSTAT_SAMPLE_INTERVAL:-30}"
pg_user="${PGSTAT_USER:-user}"
pg_db="${PGSTAT_DB:-test}"

find_container() {
  docker ps -q --filter "ancestor=$1" | head -n 1
}

psql_in() {
  docker exec "$1" psql -U "$pg_user" -d "$pg_db" -v ON_ERROR_STOP=1 -At -c "$2" 2>/dev/null
}

snapshot() {
  cid="$1"
  out="$2"
  {
    echo "# taken: $(date -u +%H:%M:%S)"
    echo "# totals: statements_total_exec_ms statements_calls db_active_time_ms db_xact_commit db_blks_hit db_blks_read db_deadlocks"
    psql_in "$cid" "
      SELECT
        COALESCE((SELECT round(sum(total_exec_time)) FROM pg_stat_statements), 0),
        COALESCE((SELECT sum(calls) FROM pg_stat_statements), 0),
        round(active_time), xact_commit, blks_hit, blks_read, deadlocks
      FROM pg_stat_database WHERE datname = current_database()"
    echo "# wait events (backends by wait_event_type)"
    psql_in "$cid" "
      SELECT state || ' ' || n FROM (
        SELECT COALESCE(wait_event_type, 'running') AS state, count(*) AS n
        FROM pg_stat_activity WHERE backend_type = 'client backend' AND pid <> pg_backend_pid()
        GROUP BY 1
      ) s ORDER BY n DESC"
    echo "# top statements: total_exec_ms calls mean_ms rows query"
    psql_in "$cid" "
      SELECT round(total_exec_time) || ' ' || calls || ' ' || round(mean_exec_time::numeric, 2) || ' ' || rows || ' ' ||
        left(regexp_replace(query, '\s+', ' ', 'g'), 140)
      FROM pg_stat_statements
      WHERE query NOT ILIKE '%pg_stat_%'
      ORDER BY total_exec_time DESC LIMIT 25"
  } > "$out.tmp" && mv "$out.tmp" "$out"
}

sample_loop() {
  dir="$1"
  image="$2"
  ready=""

  while :; do
    cid="$(find_container "$image")"
    if [ -z "$cid" ]; then
      if [ -n "$ready" ]; then
        echo "$(date -u +%H:%M:%S) container gone" >> "$dir/sampler.log"
        exit 0
      fi
      sleep 5
      continue
    fi

    if [ -z "$ready" ]; then
      if psql_in "$cid" "CREATE EXTENSION IF NOT EXISTS pg_stat_statements" >/dev/null; then
        ready=1
        echo "$(date -u +%H:%M:%S) sampling container $cid" >> "$dir/sampler.log"
      else
        sleep 5
        continue
      fi
    fi

    snapshot "$cid" "$dir/pgstat-$(date -u +%H%M%S).txt"
    sleep "$interval"
  done
}

case "${1:-}" in
  start)
    dir="$2"
    image="${3:-postgres:17-alpine}"
    mkdir -p "$dir"
    sample_loop "$dir" "$image" > /dev/null 2>&1 &
    echo $! > "$dir/sampler.pid"
    echo "pgstat sampler started (pid $!, every ${interval}s from $image) -> $dir"
    ;;
  stop)
    dir="$2"
    if [ -f "$dir/sampler.pid" ]; then
      pid="$(cat "$dir/sampler.pid")"
      pkill -P "$pid" 2>/dev/null || true
      kill "$pid" 2>/dev/null || true
      rm -f "$dir/sampler.pid"
    fi
    ;;
  summarize)
    dir="$2"
    last="$(ls "$dir"/pgstat-*.txt 2>/dev/null | tail -n 1)"
    if [ -z "$last" ]; then
      echo "no pg_stat snapshots captured"
      exit 0
    fi
    count="$(ls "$dir"/pgstat-*.txt | wc -l | tr -d ' ')"
    echo "pg_stat snapshots: $count, last one at $(awk '/^# taken:/ { print $3 }' "$last")"
    awk '
      /^# totals:/ { getline; split($0, t, "|")
        printf "total statement exec time: %.1fs over %d calls\n", t[1] / 1000, t[2]
        printf "database active time: %.1fs, commits: %d, buffer hits: %d, reads: %d, deadlocks: %d\n", t[3] / 1000, t[4], t[5], t[6], t[7]
        next }
      /^# wait events/ { print "backends by wait state at the last snapshot:"; mode = "wait"; next }
      /^# top statements/ { print "top statements by total exec time (ms, calls, mean ms, rows):"; mode = "stmt"; next }
      mode == "wait" { printf "  %s\n", $0 }
      mode == "stmt" { printf "  %8s %7s %8s %8s  %s\n", $1, $2, $3, $4, substr($0, length($1 " " $2 " " $3 " " $4 " ") + 1) }
    ' "$last"
    ;;
  *)
    echo "usage: pgstat-sampler.sh start <dir> [image] | stop <dir> | summarize <dir>" >&2
    exit 2
    ;;
esac
