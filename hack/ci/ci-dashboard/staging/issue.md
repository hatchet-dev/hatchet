# CI Health Dashboard

_Window: last 14 days (trend + pass rate) · tables: last 24h · updated 2026-08-10T07:06:01Z · auto-generated, do not edit by hand._

**Gating-CI pass rate** — PR: 82% (1716/2091) · main: 62% (57/92)

## Gating-CI pass-rate trend

```mermaid
xychart-beta
  title "Gating-CI pass rate (%) per day"
  x-axis [27, 28, 29, 30, 31, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
  y-axis "pass rate %" 0 --> 100
  line "CI" [86, 84, 84, 79, 81, 100, 93, 78, 90, 82, 65, 72, 94, 88, 96]
  line "main" [41, 60, 78, 100, 38, 38, 100, 67, 100, 50, 17, 100, 100, 50, 50]
```

_X-axis = day of month (Jul 27 → Aug 10). Two lines: **CI** (PR gating-CI runs, generally the upper line) and **main** (post-merge main runs, lower). Y-axis = % of that day's gating-CI runs that passed._

## Top 10 failing jobs (last 24h)

| # | job | workflow | fails | recovered | runs | fail rate | flaky? | scope | cause |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | `cypress` | frontend / app | 2 | 0 | 8 | 25% | flaky | PR | **infra/CI** — Cypress job: engine on :8733 not ready (connection refused) |
| 2 | `generate` | test | 2 | 0 | 10 | 20% | flaky | PR | **product bug** — Go SDK codegen: standalone_constructors_go127.go uses invalid type parameters |
| 3 | `e2e-pgmq` | test | 2 | 0 | 10 | 20% | flaky | main | **flaky test** — TestMultipleEvictionCycle intermittent in e2e-pgmq |
| 4 | `authdisabled` | build | 2 | 0 | 12 | 17% | flaky | PR | **infra/CI** — Docker frontend build fails on outdated pnpm-lock.yaml (frozen-lockfile) |
| 5 | `lite-amd` | build | 2 | 0 | 12 | 17% | flaky | PR | **infra/CI** — Docker frontend build fails on outdated pnpm-lock.yaml (frozen-lockfile) |
| 6 | `lite-arm` | build | 2 | 0 | 12 | 17% | flaky | PR | **infra/CI** — Docker frontend build fails on outdated pnpm-lock.yaml (frozen-lockfile) |
| 7 | `dashboard-amd` | build | 2 | 0 | 12 | 17% | flaky | PR | **infra/CI** — Docker frontend build fails on outdated pnpm-lock.yaml (frozen-lockfile) |
| 8 | `dashboard-arm` | build | 2 | 0 | 12 | 17% | flaky | PR | **infra/CI** — Docker frontend build fails on outdated pnpm-lock.yaml (frozen-lockfile) |
| 9 | `build` | frontend / app | 1 | 0 | 8 | 12% | flaky | PR | **infra/CI** — pnpm-lock.yaml out of sync with package.json (frozen-lockfile) |
| 10 | `lint` | frontend / app | 1 | 0 | 8 | 12% | flaky | PR | **infra/CI** — pnpm-lock.yaml out of sync with package.json (frozen-lockfile) |

## Top 10 failing tests (last 24h)

| # | test | job | fails | runs | fail rate | flaky? | scope | cause |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | `TestMultipleEvictionCycle` | `e2e-pgmq` | 2 | 10 | 20% | flaky | main | **flaky test** — TestMultipleEvictionCycle intermittent in e2e-pgmq |
| 2 | `(unparsed)` | `lite-amd` | 2 | 12 | 17% | flaky | PR | **infra/CI** — Docker frontend build fails on outdated pnpm-lock.yaml (frozen-lockfile) |
| 3 | `(unparsed)` | `dashboard-amd` | 2 | 12 | 17% | flaky | PR | **infra/CI** — Docker frontend build fails on outdated pnpm-lock.yaml (frozen-lockfile) |
| 4 | `(unparsed)` | `dashboard-arm` | 2 | 12 | 17% | flaky | PR | **infra/CI** — Docker frontend build fails on outdated pnpm-lock.yaml (frozen-lockfile) |
| 5 | `(unparsed)` | `build` | 1 | 8 | 12% | flaky | PR | **infra/CI** — pnpm-lock.yaml out of sync with package.json (frozen-lockfile) |
| 6 | `(unparsed)` | `lint` | 1 | 8 | 12% | flaky | PR | **infra/CI** — pnpm-lock.yaml out of sync with package.json (frozen-lockfile) |
| 7 | `(unparsed)` | `test` | 1 | 8 | 12% | flaky | PR | **infra/CI** — pnpm-lock.yaml out of sync with package.json (frozen-lockfile) |
| 8 | `(unparsed)` | `cypress` | 1 | 8 | 12% | flaky | PR | **infra/CI** — Cypress job: engine on :8733 not ready (connection refused) |
| 9 | `(unparsed)` | `cypress` | 1 | 8 | 12% | flaky | PR | **flaky test** — Cypress E2E: app/engine startup races and DOM timing flakes |
| 10 | `(unparsed)` | `generate` | 1 | 10 | 10% | flaky | PR | **product bug** — Go SDK codegen: standalone_constructors_go127.go uses invalid type parameters |

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
