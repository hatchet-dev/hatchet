# CI Health Dashboard

_Window: last 14 days (trend + pass rate) · tables: last 24h · updated 2026-06-29T07:08:15Z · auto-generated, do not edit by hand._

**Gating-CI pass rate** — PR: 78% (1232/1571) · main: 61% (76/124)

## Gating-CI pass-rate trend

```mermaid
xychart-beta
  title "Gating-CI pass rate (%) per day"
  x-axis [15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29]
  y-axis "pass rate %" 0 --> 100
  line "CI" [77, 53, 76, 80, 77, 81, 86, 83, 82, 84, 80, 87, 79, 100, 83]
  line "main" [56, 33, 54, 83, 70, 67, 67, 70, 50, 33, 40, 82, 0, 0, 0]
```

_X-axis = day of month (Jun 15 → Jun 29). Two lines: **CI** (PR gating-CI runs, generally the upper line) and **main** (post-merge main runs, lower). Y-axis = % of that day's gating-CI runs that passed._

## Top 10 failing jobs (last 24h)

| # | job | workflow | fails | recovered | runs | fail rate | flaky? | scope | cause |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | `cypress` | frontend / app | 1 | 0 | 1 | 100% | deterministic | PR | **dependency** — Dependabot npm-deps bump broke Cypress UI assertions/timeouts on frontend/app |
| 2 | `test` | ruby | 1 | 0 | 2 | 50% | flaky | PR | **flaky test** — ruby non_retryable e2e misses retrying event (expected 1, got 0) |
| 3 | `test` | python | 1 | 0 | 2 | 50% | flaky | PR | **flaky test** — conditions test_waits races on skip vs random_number event ordering |
| 4 | `unit` | test | 1 | 0 | 3 | 33% | flaky | PR | **flaky test** — msgqueue TestMsgIdBufferMemoryLeak times out sending messages under CI load |

## Top 10 failing tests (last 24h)

| # | test | job | fails | runs | fail rate | flaky? | scope | cause |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | `(unparsed)` | `cypress` | 1 | 1 | 100% | deterministic | PR | **dependency** — Dependabot npm-deps bump broke Cypress UI assertions/timeouts on frontend/app |
| 2 | `./non_retryable/test_no_retry_spec.rb:7` | `test` | 1 | 2 | 50% | flaky | PR | **flaky test** — ruby non_retryable e2e misses retrying event (expected 1, got 0) |
| 3 | `examples/conditions/test_conditions.py::test_waits` | `test` | 1 | 2 | 50% | flaky | PR | **flaky test** — conditions test_waits races on skip vs random_number event ordering |
| 4 | `TestMsgIdBufferMemoryLeak` | `unit` | 1 | 3 | 33% | flaky | PR | **flaky test** — msgqueue TestMsgIdBufferMemoryLeak times out sending messages under CI load |

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
