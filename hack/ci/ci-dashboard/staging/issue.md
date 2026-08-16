# CI Health Dashboard

_Window: last 14 days (trend + pass rate) · tables: last 24h · updated 2026-08-16T07:04:23Z · auto-generated, do not edit by hand._

**Gating-CI pass rate** — PR: 81% (1538/1905) · main: 49% (43/88)

## Gating-CI pass-rate trend

```mermaid
xychart-beta
  title "Gating-CI pass rate (%) per day"
  x-axis [2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15]
  y-axis "pass rate %" 0 --> 100
  line "CI" [96, 78, 90, 81, 65, 72, 94, 88, 84, 86, 74, 88, 82, 61]
  line "main" [100, 67, 100, 50, 17, 100, 100, 50, 33, 35, 25, 50, 60, 60]
```

_X-axis = day of month (Aug 02 → Aug 15). Two lines: **CI** (PR gating-CI runs, generally the upper line) and **main** (post-merge main runs, lower). Y-axis = % of that day's gating-CI runs that passed._

## Top 10 failing jobs (last 24h)

| # | job | workflow | fails | recovered | runs | fail rate | flaky? | scope | cause |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | `e2e-pgmq` | test | 3 | 0 | 4 | 75% | flaky | PR | **infra/CI** — E2E job timed out waiting for Hatchet engine/API to become ready in CI |
| 2 | `e2e` | test | 3 | 0 | 4 | 75% | flaky | PR | **infra/CI** — E2E job timed out waiting for Hatchet engine/API to become ready in CI |
| 3 | `unit` | test | 3 | 0 | 4 | 75% | flaky | PR | **product bug** — GetTaskStats SQL UNION column mismatch in repository layer |
| 4 | `lint` | ruby | 3 | 0 | 4 | 75% | flaky | PR | **infra/CI** — Ruby generated bindings check failed (codegen drift) |
| 5 | `lint` | typescript | 1 | 0 | 4 | 25% | flaky | PR | **infra/CI** — TypeScript generated bindings check failed (codegen drift) |

## Top 10 failing tests (last 24h)

| # | test | job | fails | runs | fail rate | flaky? | scope | cause |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | `(unparsed)` | `e2e-pgmq` | 3 | 4 | 75% | flaky | PR | **infra/CI** — E2E job timed out waiting for Hatchet engine/API to become ready in CI |
| 2 | `(unparsed)` | `e2e` | 3 | 4 | 75% | flaky | PR | **infra/CI** — E2E job timed out waiting for Hatchet engine/API to become ready in CI |
| 3 | `(unparsed)` | `lint` | 3 | 4 | 75% | flaky | PR | **infra/CI** — Python generated bindings check failed (codegen drift) |
| 4 | `(unparsed)` | `lint` | 3 | 4 | 75% | flaky | PR | **infra/CI** — Ruby generated bindings check failed (codegen drift) |
| 5 | `TestGetTaskStats` | `unit` | 3 | 4 | 75% | flaky | PR | **product bug** — GetTaskStats SQL UNION column mismatch in repository layer |
| 6 | `TestGetTaskStats/duplicate_strategies_count_one_running_attempt` | `unit` | 3 | 4 | 75% | flaky | PR | **product bug** — GetTaskStats SQL UNION column mismatch in repository layer |
| 7 | `(unparsed)` | `lint` | 3 | 4 | 75% | flaky | PR | **infra/CI** — TypeScript generated bindings check failed (codegen drift) |

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
