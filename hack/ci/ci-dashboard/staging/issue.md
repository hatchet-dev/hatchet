# CI Health Dashboard

_Window: last 14 days (trend + pass rate) · tables: last 24h · updated 2026-08-09T07:03:51Z · auto-generated, do not edit by hand._

**Gating-CI pass rate** — PR: 82% (1700/2076) · main: 63% (58/92)

## Gating-CI pass-rate trend

```mermaid
xychart-beta
  title "Gating-CI pass rate (%) per day"
  x-axis [26, 27, 28, 29, 30, 31, 1, 2, 3, 4, 5, 6, 7, 8]
  y-axis "pass rate %" 0 --> 100
  line "CI" [85, 87, 84, 84, 79, 81, 100, 93, 78, 90, 82, 65, 72, 94]
  line "main" [75, 41, 60, 78, 100, 38, 38, 100, 67, 100, 50, 17, 100, 100]
```

_X-axis = day of month (Jul 26 → Aug 08). Two lines: **CI** (PR gating-CI runs, generally the upper line) and **main** (post-merge main runs, lower). Y-axis = % of that day's gating-CI runs that passed._

## Top 10 failing jobs (last 24h)

| # | job | workflow | fails | recovered | runs | fail rate | flaky? | scope | cause |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | `e2e-pgmq` | test | 1 | 0 | 2 | 50% | flaky | PR | **flaky test** — TestMultipleEvictionCycle is timing-sensitive and intermittently fails in e2e-pgmq |

## Top 10 failing tests (last 24h)

| # | test | job | fails | runs | fail rate | flaky? | scope | cause |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | `TestMultipleEvictionCycle` | `e2e-pgmq` | 1 | 2 | 50% | flaky | PR | **flaky test** — TestMultipleEvictionCycle is timing-sensitive and intermittently fails in e2e-pgmq |

## Recent CI-health wins (`ci-health`)

**Recently merged**

- https://github.com/hatchet-dev/hatchet/pull/4239
- https://github.com/hatchet-dev/hatchet/pull/4238
- https://github.com/hatchet-dev/hatchet/pull/4218
- https://github.com/hatchet-dev/hatchet/pull/4213
- https://github.com/hatchet-dev/hatchet/pull/4165

**Open**

_No open `ci-health` PRs yet._

---
_Trend and pass-rate totals cover the last 14 days; job/test tables cover the last 24h._ **fails** = gating runs where the job/test failed · **recovered** = failed on a first attempt but passed on re-run (a flakiness signal) · **runs** = total gating runs of that workflow · **fail rate** = fails ÷ runs · **flaky** = recovered on re-run or intermittent across runs; **deterministic** = fails every time it runs · **scope** = whether failures were seen on PR, main, or main + PR.
