<!-- ci-health-dashboard:v1 -->
# CI Health Dashboard

_Window: last 14 days · updated 2026-06-15T20:45:10Z · auto-generated, do not edit by hand._

**Gating-CI pass rate** — PR: 72% (1534/2141) · main: 43% (72/168)

## Gating-CI pass-rate trend

```mermaid
xychart-beta
  title "Gating-CI pass rate (%) per day"
  x-axis [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 15]
  y-axis "pass rate %" 0 --> 100
  line "CI" [62, 68, 69, 69, 74, 77, 60, 68, 71, 74, 73, 71, 92, 80]
  line "main" [33, 41, 0, 40, 40, 36, 38, 40, 43, 0, 67, 50, 50, 55]
```

_X-axis = day of month (Jun 01 → Jun 15). Two lines: **CI** (PR gating-CI runs, generally the upper line) and **main** (post-merge main runs, lower). Y-axis = % of that day's gating-CI runs that passed._

## Top 10 failing jobs

| # | job | workflow | fails | recovered | runs | fail rate | flaky? | scope | cause |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | `load-online-migrate` | test | 182 | 0 | 370 | 49% | flaky | main + PR | **infra/CI** — Engine gRPC not ready on [::1]:7077 before load test worker registration |
| 2 | `old-engine-new-sdk` | typescript | 148 | 3 | 267 | 55% | flaky | main + PR | **product bug** — bulk-replay-e2e retry count assertion fails in old-engine-new-sdk TypeScript e2e |
| 3 | `generate` | test | 125 | 0 | 370 | 34% | flaky | main + PR | **infra/CI** — generate job Check-for-diff fails: committed generated code out of sync with source |
| 4 | `e2e-pgmq` | test | 93 | 1 | 370 | 25% | flaky | main + PR | **flaky test** — TestDurableErrorOnErrorInChild intermittently fails in e2e-pgmq |
| 5 | `e2e` | test | 64 | 4 | 370 | 17% | flaky | main + PR | **timeout** — TestEvictableTaskRestoreCompletes exceeds ~300s budget in e2e job |
| 6 | `old-engine-new-sdk` | python | 62 | 1 | 280 | 22% | flaky | main + PR | **product bug** — batch_assign pytest fails with FailedTaskRunExceptionGroup in old-engine-new-sdk matrix |
| 7 | `old-engine-new-sdk` | ruby | 38 | 0 | 113 | 34% | flaky | PR | **unknown** — Git fetch output noise; actual Ruby setup failure not captured in sample |
| 8 | `load-pgbouncer` | test | 30 | 2 | 370 | 8% | flaky | main + PR | **timeout** — TestLoadCLI parent fails when DAG subtest times out in pgbouncer load job |
| 9 | `load-deadlock` | test | 28 | 1 | 370 | 8% | flaky | main + PR | **flaky test** — Durable events listener reconnect timing race under deadlock instrumentation |
| 10 | `cypress` | frontend / app | 28 | 0 | 169 | 17% | flaky | PR | **flaky test** — Cypress auth/08-tenant-invite-decline.cy.ts intermittently fails (sample is shell trace noise) |

## Top 10 failing tests

| # | test | job | fails | runs | fail rate | flaky? | scope | cause |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | `(unparsed)` | `generate` | 115 | 370 | 31% | flaky | main + PR | **infra/CI** — generate job Check-for-diff fails: committed generated code out of sync with source |
| 2 | `bulk-replay-e2e › bulk replays matching runs and increments retry count` | `old-engine-new-sdk` | 104 | 267 | 39% | flaky | main + PR | **product bug** — bulk-replay-e2e retry count assertion fails in old-engine-new-sdk TypeScript e2e |
| 3 | `(unparsed)` | `load-online-migrate` | 85 | 370 | 23% | flaky | main + PR | **infra/CI** — Engine gRPC not ready on [::1]:7077 before load test worker registration |
| 4 | `(unparsed)` | `load-online-migrate` | 55 | 370 | 15% | flaky | PR | **product bug** — load-online-migrate panics with generic load test failed after engine/worker setup issues |
| 5 | `(unparsed)` | `load-online-migrate` | 37 | 370 | 10% | flaky | main + PR | **infra/CI** — Engine gRPC not ready on localhost:7077 before load test worker registration |
| 6 | `TestLoadCLI` | `load-pgbouncer` | 33 | 370 | 9% | flaky | main + PR | **timeout** — TestLoadCLI parent fails when DAG subtest times out in pgbouncer load job |
| 7 | `TestLoadCLI/test_with_DAG` | `load-pgbouncer` | 31 | 370 | 8% | flaky | main + PR | **timeout** — TestLoadCLI/test_with_DAG hits 340s test timeout in pgbouncer load job |
| 8 | `TestDurableEventsListenerDeliversEventAfterReconnectDuringRetryBackoff` | `load-deadlock` | 30 | 370 | 8% | flaky | main + PR | **flaky test** — Durable events listener reconnect timing race under deadlock instrumentation |
| 9 | `TestDurableErrorOnErrorInChild` | `e2e-pgmq` | 28 | 370 | 8% | flaky | main + PR | **flaky test** — TestDurableErrorOnErrorInChild intermittently fails in e2e-pgmq |
| 10 | `(unparsed)` | `cypress` | 27 | 169 | 16% | flaky | PR | **flaky test** — Cypress auth/08-tenant-invite-decline.cy.ts intermittently fails (sample is shell trace noise) |

## Recent CI-health wins (`ci-health`)

**Recently merged**

- https://github.com/hatchet-dev/hatchet/pull/4165
- https://github.com/hatchet-dev/hatchet/pull/4159
- https://github.com/hatchet-dev/hatchet/pull/4156
- https://github.com/hatchet-dev/hatchet/pull/4146
- https://github.com/hatchet-dev/hatchet/pull/4145

**Open**

_No open `ci-health` PRs yet._

---
_All counts cover the window above (last 14 days)._ **fails** = gating runs where the job/test failed · **recovered** = failed on a first attempt but passed on re-run (a flakiness signal) · **runs** = total gating runs of that workflow · **fail rate** = fails ÷ runs · **flaky** = recovered on re-run or intermittent across runs; **deterministic** = fails every time it runs · **scope** = whether failures were seen on PR, main, or main + PR.
