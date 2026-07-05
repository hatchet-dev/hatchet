# CI Health Dashboard

_Window: last 14 days (trend + pass rate) · tables: last 24h · updated 2026-07-05T07:06:05Z · auto-generated, do not edit by hand._

**Gating-CI pass rate** — PR: 82% (1555/1895) · main: 63% (62/99)

## Gating-CI pass-rate trend

```mermaid
xychart-beta
  title "Gating-CI pass rate (%) per day"
  x-axis [21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 1, 2, 3, 4]
  y-axis "pass rate %" 0 --> 100
  line "CI" [86, 83, 82, 84, 80, 87, 79, 100, 79, 82, 78, 85, 76, 95]
  line "main" [70, 70, 50, 33, 40, 82, 100, 100, 90, 50, 67, 53, 67, 67]
```

_X-axis = day of month (Jun 21 → Jul 04). Two lines: **CI** (PR gating-CI runs, generally the upper line) and **main** (post-merge main runs, lower). Y-axis = % of that day's gating-CI runs that passed._

## Top 10 failing jobs (last 24h)

_No failing jobs._

## Top 10 failing tests (last 24h)

_No failing tests parsed._

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
