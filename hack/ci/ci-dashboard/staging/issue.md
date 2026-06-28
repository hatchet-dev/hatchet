# CI Health Dashboard

_Window: last 14 days (trend + pass rate) · tables: last 24h · updated 2026-06-28T07:07:06Z · auto-generated, do not edit by hand._

**Gating-CI pass rate** — PR: 78% (1240/1579) · main: 61% (76/124)

## Gating-CI pass-rate trend

```mermaid
xychart-beta
  title "Gating-CI pass rate (%) per day"
  x-axis [15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27]
  y-axis "pass rate %" 0 --> 100
  line "CI" [79, 53, 76, 80, 77, 81, 86, 83, 82, 84, 80, 87, 79]
  line "main" [56, 33, 54, 83, 70, 67, 67, 70, 50, 33, 40, 82, 0]
```

_X-axis = day of month (Jun 15 → Jun 27). Two lines: **CI** (PR gating-CI runs, generally the upper line) and **main** (post-merge main runs, lower). Y-axis = % of that day's gating-CI runs that passed._

## Top 10 failing jobs (last 24h)

| # | job | workflow | fails | recovered | runs | fail rate | flaky? | scope | cause |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | `test` | python | 1 | 0 | 2 | 50% | flaky | PR | **flaky test** — conditions test_waits: expected skipped but got random_number output (timing/race) |
| 2 | `unit` | test | 1 | 0 | 3 | 33% | flaky | main | **flaky test** — interval jitter timing exceeded 85ms bound on loaded CI runner |
| 3 | `generate` | test | 1 | 0 | 3 | 33% | flaky | PR | **infra/CI** — generate job: git diff after codegen (examples/go/scheduled/main.go drift) |
| 4 | `lint` | lint all | 1 | 0 | 3 | 33% | flaky | PR | **infra/CI** — pre-commit sync-python-changelog hook drift on PR (python.mdx not updated) |

## Top 10 failing tests (last 24h)

| # | test | job | fails | runs | fail rate | flaky? | scope | cause |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | `examples/conditions/test_conditions.py::test_waits` | `test` | 1 | 2 | 50% | flaky | PR | **flaky test** — conditions test_waits: expected skipped but got random_number output (timing/race) |
| 2 | `TestInterval_RunInterval_WithJitter` | `unit` | 1 | 3 | 33% | flaky | main | **flaky test** — interval jitter timing exceeded 85ms bound on loaded CI runner |
| 3 | `(unparsed)` | `generate` | 1 | 3 | 33% | flaky | PR | **infra/CI** — generate job: git diff after codegen (examples/go/scheduled/main.go drift) |
| 4 | `TestDurableEventsListenerDeliversEventAfterReconnectDuringRetryBackoff` | `load-deadlock` | 1 | 3 | 33% | flaky | PR | **flaky test** — Durable events listener reconnect test: event not delivered within backoff window |
| 5 | `(unparsed)` | `lint` | 1 | 3 | 33% | flaky | PR | **infra/CI** — pre-commit sync-python-changelog hook drift on PR (python.mdx not updated) |

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
