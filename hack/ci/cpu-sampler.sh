#!/bin/sh
# Samples runner CPU pressure while a job step runs, so a latency failure can be
# read against what the machine was doing. Busy and steal time from /proc/stat
# are the signals: busy near 100 means the runner was saturated, steal above a
# few percent means the host took CPU away from it. Per process CPU comes from
# /proc/<pid>/stat deltas grouped by command name, so it is the share of all
# CPUs each program used during the interval rather than a lifetime average.
#
# Usage:
#   cpu-sampler.sh start <log>                       sample in the background, pid in <log>.pid
#   cpu-sampler.sh stop <log>                        stop sampling
#   cpu-sampler.sh summarize <log>                   print a summary of the samples
#   cpu-sampler.sh comment <log> [label] [run url]   markdown with charts for a PR comment
#
# Linux only (reads /proc). Samples every 2 seconds by default (CPU_SAMPLE_INTERVAL).

set -u

interval="${CPU_SAMPLE_INTERVAL:-2}"
top_processes="${CPU_SAMPLE_PROCESSES:-6}"

read_cpu() {
  # user nice system idle iowait irq softirq steal
  awk '/^cpu / { print $2, $3, $4, $5, $6, $7, $8, $9 }' /proc/stat
}

# Prints "comm ticks" per command name, summing user and system time over all
# processes with that name. The comm field is parenthesised and can contain
# spaces, so the line is split on the last closing parenthesis.
read_procs() {
  for f in /proc/[0-9]*/stat; do
    cat "$f" 2>/dev/null
  done | awk -F '[()]' 'NF >= 3 {
    # $2 is the command name; the last field holds the counters, where utime
    # is field 12 and stime field 13 after the leading space
    n = split($NF, fields, " ")
    if (n >= 13) ticks[$2] += fields[12] + fields[13]
  }
  END { for (c in ticks) print c, ticks[c] }'
}

sample_loop() {
  log="$1"
  prev_procs="$log.procs.prev"
  cur_procs="$log.procs.cur"

  {
    echo "# host: $(uname -srm)"
    echo "# cpus: $(nproc)"
    if [ -r /sys/fs/cgroup/cpu.max ]; then
      echo "# cgroup cpu.max: $(cat /sys/fs/cgroup/cpu.max)"
    fi
    echo "# interval: ${interval}s"
    echo "# columns: time load1 load5 busy% user% system% iowait% steal% mem_available_mb process=cpu%..."
    echo "# busy% is 100 minus idle; process cpu% is that command's share of all CPUs in the interval"
  } >> "$log"

  prev="$(read_cpu)"
  read_procs > "$prev_procs"

  while :; do
    sleep "$interval"

    cur="$(read_cpu)"
    read_procs > "$cur_procs"

    # fields 1-8 are the previous counters, 9-16 the current ones
    cpu_line="$(echo "$prev $cur" | awk '{
      before = $1 + $2 + $3 + $4 + $5 + $6 + $7 + $8
      after = $9 + $10 + $11 + $12 + $13 + $14 + $15 + $16
      total = after - before
      if (total <= 0) total = 1
      user = ($9 - $1 + $10 - $2) * 100 / total
      sys = ($11 - $3 + $14 - $6 + $15 - $7) * 100 / total
      idle = ($12 - $4) * 100 / total
      iowait = ($13 - $5) * 100 / total
      steal = ($16 - $8) * 100 / total
      printf "%.1f %.1f %.1f %.1f %.1f %d", 100 - idle, user, sys, iowait, steal, total
    }')"
    total_ticks="${cpu_line##* }"
    cpu_line="${cpu_line% *}"

    procs="$(awk -v total="$total_ticks" -v limit="$top_processes" '
      NR == FNR { prev[$1] = $2; next }
      { delta = $2 - prev[$1]; if (delta > 0) pct[$1] = delta * 100 / total }
      END {
        n = 0
        for (c in pct) { n++; names[n] = c }
        # selection sort, descending by share
        for (i = 1; i <= n; i++)
          for (j = i + 1; j <= n; j++)
            if (pct[names[j]] > pct[names[i]]) { t = names[i]; names[i] = names[j]; names[j] = t }
        for (i = 1; i <= n && i <= limit; i++) {
          gsub(/[ =]/, "_", names[i])
          printf "%s%s=%.1f", (i > 1 ? " " : ""), names[i], pct[names[i]]
        }
      }' "$prev_procs" "$cur_procs")"

    prev="$cur"
    mv "$cur_procs" "$prev_procs"

    loads="$(awk '{ print $1, $2 }' /proc/loadavg)"
    mem="$(awk '/MemAvailable/ { printf "%d", $2 / 1024 }' /proc/meminfo)"

    echo "$(date -u +%H:%M:%S) $loads $cpu_line $mem $procs" >> "$log"
  done
}

# Prints "comm mean_pct" for the commands with the highest mean share over the
# run, highest first, at most $1 of them.
top_commands() {
  limit="$1"
  log="$2"
  awk -v limit="$limit" '!/^#/ {
    n++
    for (i = 10; i <= NF; i++) {
      split($i, kv, "=")
      sum[kv[1]] += kv[2]
    }
  }
  END {
    if (n == 0) exit
    m = 0
    for (c in sum) { m++; names[m] = c }
    for (i = 1; i <= m; i++)
      for (j = i + 1; j <= m; j++)
        if (sum[names[j]] > sum[names[i]]) { t = names[i]; names[i] = names[j]; names[j] = t }
    for (i = 1; i <= m && i <= limit; i++) printf "%s %.1f\n", names[i], sum[names[i]] / n
  }' "$log"
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
      rm -f "$log.pid" "$log.procs.prev" "$log.procs.cur"
    fi
    ;;
  summarize)
    log="$2"
    if [ ! -s "$log" ]; then
      echo "no cpu samples recorded"
      exit 0
    fi
    grep '^#' "$log" | grep -v '^# columns'
    awk '!/^#/ {
      n++
      busy += $4; steal += $8; iowait += $7
      if ($2 > load1max) load1max = $2
      if ($8 > stealmax) stealmax = $8
      if ($4 > busymax) busymax = $4
      if ($4 > 90) saturated++
      if ($8 > 5) stolen++
    }
    END {
      if (n == 0) { print "no cpu samples recorded"; exit }
      printf "samples: %d\n", n
      printf "busy%%: mean %.1f, max %.1f, samples over 90%% busy: %d (%.0f%% of the run)\n", busy / n, busymax, saturated, saturated * 100 / n
      printf "steal%%: mean %.1f, max %.1f, samples over 5%% steal: %d (%.0f%% of the run)\n", steal / n, stealmax, stolen, stolen * 100 / n
      printf "iowait%%: mean %.1f\n", iowait / n
      printf "load1: max %.1f\n", load1max
    }' "$log"
    echo "top processes by mean share of all CPUs:"
    top_commands 8 "$log" | awk '{ printf "  %-24s %5.1f%%\n", $1, $2 }'
    ;;
  comment)
    # Markdown for a pull request comment: the summary plus two Mermaid charts,
    # runner pressure (busy, steal) and the top processes, which GitHub renders
    # inline. Mermaid charts get unreadable past a few dozen points, so the
    # samples are averaged into at most 40 buckets.
    log="$2"
    label="${3:-load test}"
    run_url="${4:-}"
    if [ ! -s "$log" ]; then
      echo "No CPU samples were recorded for the $label run."
      exit 0
    fi
    echo "### Runner CPU during the failed $label run"
    echo
    echo "Busy is 100 minus idle across all CPUs: near 100 means the runner was saturated. Steal is CPU the host took away from the runner: above a few percent means it was oversubscribed. Either way the latency gate measured the runner, not the scheduler."
    echo
    echo '```'
    "$0" summarize "$log"
    echo '```'
    echo
    tops="$(top_commands 5 "$log" | awk '{ printf "%s%s", (NR > 1 ? "," : ""), $1 }')"
    awk -v tops="$tops" '
    BEGIN { ntop = split(tops, top, ",") }
    !/^#/ {
      n++; t[n] = $1; busy[n] = $4; steal[n] = $8
      for (i = 10; i <= NF; i++) { split($i, kv, "="); share[n, kv[1]] = kv[2] }
    }
    function series(name, buckets, per,    b, lo, hi, i, sum, count, key, out) {
      out = ""
      for (b = 0; b < buckets; b++) {
        lo = int(b * per) + 1; hi = int((b + 1) * per); if (hi < lo) hi = lo; if (hi > n) hi = n
        sum = 0; count = 0
        for (i = lo; i <= hi; i++) {
          if (name == "busy") sum += busy[i]
          else if (name == "steal") sum += steal[i]
          else { key = i SUBSEP name; sum += (key in share ? share[key] : 0) }
          count++
        }
        out = out (b ? ", " : "") sprintf("%.1f", sum / count)
      }
      return out
    }
    function axis(buckets, per,    b, i, out) {
      out = ""
      for (b = 0; b < buckets; b++) { i = int(b * per) + 1; out = out (b ? ", " : "") "\"" t[i] "\"" }
      return out
    }
    END {
      if (n == 0) exit
      buckets = n < 40 ? n : 40
      per = n / buckets
      printf "```mermaid\nxychart-beta\n    title \"Runner pressure (%d samples)\"\n", n
      printf "    x-axis \"sample time\" [%s]\n    y-axis \"percent of all CPUs\" 0 --> 100\n", axis(buckets, per)
      printf "    line \"busy\" [%s]\n    line \"steal\" [%s]\n```\n\n", series("busy", buckets, per), series("steal", buckets, per)
      if (ntop == 0) exit
      printf "```mermaid\nxychart-beta\n    title \"Top processes, share of all CPUs\"\n"
      printf "    x-axis \"sample time\" [%s]\n    y-axis \"percent of all CPUs\" 0 --> 100\n", axis(buckets, per)
      for (k = 1; k <= ntop; k++) printf "    line \"%s\" [%s]\n", top[k], series(top[k], buckets, per)
      printf "```\n"
      printf "\nProcess lines, in order: "
      for (k = 1; k <= ntop; k++) printf "%s%s", (k > 1 ? ", " : ""), top[k]
      printf "\n"
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
