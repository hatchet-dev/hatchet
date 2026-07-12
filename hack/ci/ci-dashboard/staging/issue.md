# CI Health Dashboard

_Window: last 14 days (trend + pass rate) · tables: last 24h · updated 2026-07-12T07:05:38Z · auto-generated, do not edit by hand._

**Gating-CI pass rate** — PR: 82% (1914/2334) · main: 64% (72/113)

## Gating-CI pass-rate trend

```mermaid
xychart-beta
  title "Gating-CI pass rate (%) per day"
  x-axis [28, 29, 30, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11]
  y-axis "pass rate %" 0 --> 100
  line "CI" [100, 79, 82, 78, 85, 76, 95, 85, 81, 84, 86, 76, 83, 96]
  line "main" [90, 90, 50, 67, 53, 67, 67, 67, 100, 67, 60, 36, 100, 100]
```

_X-axis = day of month (Jun 28 → Jul 11). Two lines: **CI** (PR gating-CI runs, generally the upper line) and **main** (post-merge main runs, lower). Y-axis = % of that day's gating-CI runs that passed._

## Top 10 failing jobs (last 24h)

| # | job | workflow | fails | recovered | runs | fail rate | flaky? | scope | cause |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | `e2e` | test | 1 | 0 | 3 | 33% | flaky | PR | **infra/CI** — e2e job timed out waiting for Hatchet engine/API readiness in CI |

## Top 10 failing tests (last 24h)

| # | test | job | fails | runs | fail rate | flaky? | scope | cause |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | `(unparsed)` | `e2e` | 1 | 3 | 33% | flaky | PR | **infra/CI** — e2e job timed out waiting for Hatchet engine/API readiness in CI |

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
