#!/bin/sh
# Samples runner CPU pressure while a job step runs, so a latency failure can be
# read against what the machine was doing. Steal time and idle time from
# /proc/stat are the signals: high steal or near-zero idle means the runner was
# CPU constrained, whatever the test binary itself was doing.
#
# Usage:
#   cpu-sampler.sh start <log>      start sampling in the background, pid in <log>.pid
#   cpu-sampler.sh stop <log>       stop sampling
#   cpu-sampler.sh summarize <log>  print a summary of the samples
#
# Linux only (reads /proc). Samples every 5 seconds by default (CPU_SAMPLE_INTERVAL).

set -u

interval="${CPU_SAMPLE_INTERVAL:-5}"

read_cpu() {
  # user nice system idle iowait irq softirq steal
  awk '/^cpu / { print $2, $3, $4, $5, $6, $7, $8, $9 }' /proc/stat
}

sample_loop() {
  log="$1"

  {
    echo "# host: $(uname -srm)"
    echo "# cpus: $(nproc)"
    if [ -r /sys/fs/cgroup/cpu.max ]; then
      echo "# cgroup cpu.max: $(cat /sys/fs/cgroup/cpu.max)"
    fi
    echo "# interval: ${interval}s"
    echo "# columns: time load1 load5 user% system% idle% iowait% steal% mem_available_mb top_processes"
  } >> "$log"

  prev="$(read_cpu)"

  while :; do
    sleep "$interval"

    cur="$(read_cpu)"
    # fields 1-8 are the previous counters, 9-16 the current ones
    line="$(echo "$prev $cur" | awk '{
      before = $1 + $2 + $3 + $4 + $5 + $6 + $7 + $8
      after = $9 + $10 + $11 + $12 + $13 + $14 + $15 + $16
      total = after - before
      if (total <= 0) total = 1
      user = ($9 - $1 + $10 - $2) * 100 / total
      sys = ($11 - $3 + $14 - $6 + $15 - $7) * 100 / total
      idle = ($12 - $4) * 100 / total
      iowait = ($13 - $5) * 100 / total
      steal = ($16 - $8) * 100 / total
      printf "%.1f %.1f %.1f %.1f %.1f", user, sys, idle, iowait, steal
    }')"
    prev="$cur"

    loads="$(awk '{ print $1, $2 }' /proc/loadavg)"
    mem="$(awk '/MemAvailable/ { printf "%d", $2 / 1024 }' /proc/meminfo)"
    top="$(ps -eo pcpu,comm --sort=-pcpu 2>/dev/null | awk 'NR > 1 && NR <= 6 { printf "%s:%s ", $2, $1 }')"

    echo "$(date -u +%H:%M:%S) $loads $line $mem $top" >> "$log"
  done
}

case "${1:-}" in
  start)
    log="$2"
    : > "$log"
    sample_loop "$log" &
    echo $! > "$log.pid"
    echo "cpu sampler started (pid $!, every ${interval}s) -> $log"
    ;;
  stop)
    log="$2"
    if [ -f "$log.pid" ]; then
      kill "$(cat "$log.pid")" 2>/dev/null || true
      rm -f "$log.pid"
    fi
    ;;
  summarize)
    log="$2"
    if [ ! -s "$log" ]; then
      echo "no cpu samples recorded"
      exit 0
    fi
    grep '^#' "$log"
    awk '!/^#/ {
      n++
      idle += $6; steal += $8; iowait += $7
      if ($2 > load1max) load1max = $2
      if ($8 > stealmax) stealmax = $8
      if ($6 < idlemin || n == 1) idlemin = $6
      if ($6 < 10) starved++
      if ($8 > 5) stolen++
    }
    END {
      if (n == 0) { print "no cpu samples recorded"; exit }
      printf "samples: %d\n", n
      printf "idle%%: mean %.1f, min %.1f, samples under 10%% idle: %d (%.0f%% of the run)\n", idle / n, idlemin, starved, starved * 100 / n
      printf "steal%%: mean %.1f, max %.1f, samples over 5%% steal: %d\n", steal / n, stealmax, stolen
      printf "iowait%%: mean %.1f\n", iowait / n
      printf "load1: max %.1f\n", load1max
    }' "$log"
    ;;
  comment)
    # Markdown for a pull request comment: the summary plus a Mermaid chart of
    # idle and steal over the run, which GitHub renders inline. Mermaid charts
    # get unreadable past a few dozen points, so the samples are averaged into
    # at most 40 buckets.
    log="$2"
    label="${3:-load test}"
    run_url="${4:-}"
    if [ ! -s "$log" ]; then
      echo "No CPU samples were recorded for the $label run."
      exit 0
    fi
    echo "### Runner CPU during the failed $label run"
    echo
    echo "Idle near zero means the runner was saturated; steal above a few percent means the host took CPU away from it. Either way the latency gate measured the runner, not the scheduler."
    echo
    echo '```'
    "$0" summarize "$log" | grep -v '^# columns'
    echo '```'
    echo
    awk '!/^#/ { n++; t[n] = $1; idle[n] = $6; steal[n] = $8; load[n] = $2 }
    END {
      if (n == 0) exit
      buckets = n < 40 ? n : 40
      per = n / buckets
      printf "```mermaid\nxychart-beta\n    title \"Runner CPU (%d samples)\"\n", n
      printf "    x-axis \"sample time\" ["
      for (b = 0; b < buckets; b++) {
        i = int(b * per) + 1
        printf "%s\"%s\"", (b ? ", " : ""), t[i]
      }
      printf "]\n    y-axis \"percent\" 0 --> 100\n"
      for (series = 1; series <= 2; series++) {
        printf "    line \"%s\" [", (series == 1 ? "idle" : "steal")
        for (b = 0; b < buckets; b++) {
          lo = int(b * per) + 1; hi = int((b + 1) * per); if (hi < lo) hi = lo; if (hi > n) hi = n
          sum = 0; count = 0
          for (i = lo; i <= hi; i++) { sum += (series == 1 ? idle[i] : steal[i]); count++ }
          printf "%s%.1f", (b ? ", " : ""), sum / count
        }
        printf "]\n"
      }
      printf "```\n"
    }' "$log"
    if [ -n "$run_url" ]; then
      echo
      echo "Raw samples are in the run's \`load-cpu-samples-*\` artifacts: $run_url"
    fi
    ;;
  *)
    echo "usage: cpu-sampler.sh start|stop|summarize|comment <log> [label] [run-url]" >&2
    exit 2
    ;;
esac
