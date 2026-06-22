# CI Health Dashboard

_Window: last 14 days (trend + pass rate) · tables: last 24h · updated 2026-06-22T07:08:26Z · auto-generated, do not edit by hand._

**Gating-CI pass rate** — PR: 73% (1212/1654) · main: 53% (80/150)

## Gating-CI pass-rate trend

```mermaid
xychart-beta
  title "Gating-CI pass rate (%) per day"
  x-axis [8, 9, 10, 11, 12, 13, 15, 16, 17, 18, 19, 20, 21, 22]
  y-axis "pass rate %" 0 --> 100
  line "CI" [67, 71, 74, 73, 71, 92, 79, 56, 76, 81, 75, 81, 86, 96]
  line "main" [40, 43, 0, 67, 50, 50, 56, 33, 54, 83, 70, 67, 67, 67]
```

_X-axis = day of month (Jun 08 → Jun 22). Two lines: **CI** (PR gating-CI runs, generally the upper line) and **main** (post-merge main runs, lower). Y-axis = % of that day's gating-CI runs that passed._

## Top 10 failing jobs (last 24h)

| # | job | workflow | fails | recovered | runs | fail rate | flaky? | scope | cause |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | `cypress` | frontend / app | 1 | 0 | 3 | 33% | flaky | PR | **dependency** — Dependabot npm-deps bump broke Cypress (sidebar/login assertions) |

## Top 10 failing tests (last 24h)

| # | test | job | fails | runs | fail rate | flaky? | scope | cause |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | `(unparsed)` | `cypress` | 1 | 3 | 33% | flaky | PR | **dependency** — Dependabot npm-deps bump broke Cypress (sidebar/login assertions) |

## Recent CI-health wins (`ci-health`)

**Recently merged**

- https://github.com/hatchet-dev/hatchet/pull/4239
- https://github.com/hatchet-dev/hatchet/pull/4238
- https://github.com/hatchet-dev/hatchet/pull/4218
- https://github.com/hatchet-dev/hatchet/pull/4213
- https://github.com/hatchet-dev/hatchet/pull/4165

**Open**

- https://github.com/hatchet-dev/hatchet/pull/4212

---
_Trend and pass-rate totals cover the last 14 days; job/test tables cover the last 24h._ **fails** = gating runs where the job/test failed · **recovered** = failed on a first attempt but passed on re-run (a flakiness signal) · **runs** = total gating runs of that workflow · **fail rate** = fails ÷ runs · **flaky** = recovered on re-run or intermittent across runs; **deterministic** = fails every time it runs · **scope** = whether failures were seen on PR, main, or main + PR.
