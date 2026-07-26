# CI Health Dashboard

_Window: last 14 days (trend + pass rate) · tables: last 24h · updated 2026-07-26T07:04:42Z · auto-generated, do not edit by hand._

**Gating-CI pass rate** — PR: 79% (1995/2536) · main: 72% (107/148)

## Gating-CI pass-rate trend

```mermaid
xychart-beta
  title "Gating-CI pass rate (%) per day"
  x-axis [11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25]
  y-axis "pass rate %" 0 --> 100
  line "CI" [92, 97, 78, 81, 84, 88, 81, 67, 81, 75, 74, 77, 69, 83, 95]
  line "main" [67, 67, 67, 89, 71, 75, 40, 40, 40, 79, 70, 68, 79, 20, 20]
```

_X-axis = day of month (Jul 11 → Jul 25). Two lines: **CI** (PR gating-CI runs, generally the upper line) and **main** (post-merge main runs, lower). Y-axis = % of that day's gating-CI runs that passed._

## Top 10 failing jobs (last 24h)

| # | job | workflow | fails | recovered | runs | fail rate | flaky? | scope | cause |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | `load` | test | 1 | 0 | 3 | 33% | flaky | PR | **flaky test** — TestMsgIdBufferMemoryLeak times out sending to msgIdBufferCh under concurrent load in CI |

## Top 10 failing tests (last 24h)

| # | test | job | fails | runs | fail rate | flaky? | scope | cause |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | `TestMsgIdBufferMemoryLeak` | `load` | 1 | 3 | 33% | flaky | PR | **flaky test** — TestMsgIdBufferMemoryLeak times out sending to msgIdBufferCh under concurrent load in CI |

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
