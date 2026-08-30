# CI Health Dashboard

_Window: last 14 days (trend + pass rate) · tables: last 24h · updated 2026-08-30T07:05:05Z · auto-generated, do not edit by hand._

**Gating-CI pass rate** — PR: 85% (2209/2595) · main: 76% (106/140)

## Gating-CI pass-rate trend

```mermaid
xychart-beta
  title "Gating-CI pass rate (%) per day"
  x-axis [16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29]
  y-axis "pass rate %" 0 --> 100
  line "CI" [67, 86, 89, 78, 84, 82, 73, 93, 88, 93, 88, 92, 87, 100]
  line "main" [82, 82, 73, 44, 78, 78, 100, 100, 81, 100, 88, 100, 62, 62]
```

_X-axis = day of month (Aug 16 → Aug 29). Two lines: **CI** (PR gating-CI runs, generally the upper line) and **main** (post-merge main runs, lower). Y-axis = % of that day's gating-CI runs that passed._

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
