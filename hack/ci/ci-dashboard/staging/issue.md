# CI Health Dashboard

_Window: last 14 days (trend + pass rate) · tables: last 24h · updated 2026-08-03T07:09:24Z · auto-generated, do not edit by hand._

**Gating-CI pass rate** — PR: 79% (1910/2407) · main: 68% (98/144)

## Gating-CI pass-rate trend

```mermaid
xychart-beta
  title "Gating-CI pass rate (%) per day"
  x-axis [20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 1, 2, 3]
  y-axis "pass rate %" 0 --> 100
  line "CI" [73, 74, 77, 69, 83, 95, 85, 87, 85, 84, 79, 81, 100, 93, 88]
  line "main" [79, 70, 68, 79, 20, 20, 75, 41, 60, 78, 100, 38, 38, 100, 100]
```

_X-axis = day of month (Jul 20 → Aug 03). Two lines: **CI** (PR gating-CI runs, generally the upper line) and **main** (post-merge main runs, lower). Y-axis = % of that day's gating-CI runs that passed._

## Top 10 failing jobs (last 24h)

| # | job | workflow | fails | recovered | runs | fail rate | flaky? | scope | cause |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | `e2e-pgmq` | test | 2 | 0 | 9 | 22% | flaky | PR | **flaky test** — TestMultipleEvictionCycle fails intermittently on e2e-pgmq eviction timing |
| 2 | `cypress` | frontend / app | 1 | 0 | 3 | 33% | flaky | PR | **flaky test** — Cypress cy.wait timed out waiting for scheduledRuns route in frontend e2e |
| 3 | `unit` | test | 1 | 0 | 9 | 11% | flaky | PR | **flaky test** — TestMsgIdBufferMemoryLeak fails intermittently on unit job buffer lifecycle timing |
| 4 | `load` | test | 1 | 0 | 9 | 11% | flaky | PR | **flaky test** — TestListenReconnectingStreamHandlesEventsAndStopsOnEOF fails intermittently on load job stream reconnect timing |
| 5 | `admin` | build | 1 | 0 | 11 | 9% | flaky | PR | **infra/CI** — Docker admin build fails when Alpine apk update hits TLS error fetching APKINDEX |

## Top 10 failing tests (last 24h)

| # | test | job | fails | runs | fail rate | flaky? | scope | cause |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | `TestMultipleEvictionCycle` | `e2e-pgmq` | 2 | 9 | 22% | flaky | PR | **flaky test** — TestMultipleEvictionCycle fails intermittently on e2e-pgmq eviction timing |
| 2 | `(unparsed)` | `cypress` | 1 | 3 | 33% | flaky | PR | **flaky test** — Cypress cy.wait timed out waiting for scheduledRuns route in frontend e2e |
| 3 | `TestMsgIdBufferMemoryLeak` | `unit` | 1 | 9 | 11% | flaky | PR | **flaky test** — TestMsgIdBufferMemoryLeak fails intermittently on unit job buffer lifecycle timing |
| 4 | `TestListenReconnectingStreamHandlesEventsAndStopsOnEOF` | `load` | 1 | 9 | 11% | flaky | PR | **flaky test** — TestListenReconnectingStreamHandlesEventsAndStopsOnEOF fails intermittently on load job stream reconnect timing |
| 5 | `(unparsed)` | `admin` | 1 | 11 | 9% | flaky | PR | **infra/CI** — Docker admin build fails when Alpine apk update hits TLS error fetching APKINDEX |

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
