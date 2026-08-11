# CI Health Dashboard

_Window: last 14 days (trend + pass rate) · tables: last 24h · updated 2026-08-11T07:08:31Z · auto-generated, do not edit by hand._

**Gating-CI pass rate** — PR: 82% (1740/2128) · main: 61% (55/90)

## Gating-CI pass-rate trend

```mermaid
xychart-beta
  title "Gating-CI pass rate (%) per day"
  x-axis [28, 29, 30, 31, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
  y-axis "pass rate %" 0 --> 100
  line "CI" [84, 84, 79, 81, 100, 93, 78, 90, 81, 65, 72, 94, 88, 84]
  line "main" [60, 78, 100, 38, 38, 100, 67, 100, 50, 17, 100, 100, 50, 33]
```

_X-axis = day of month (Jul 28 → Aug 10). Two lines: **CI** (PR gating-CI runs, generally the upper line) and **main** (post-merge main runs, lower). Y-axis = % of that day's gating-CI runs that passed._

## Top 10 failing jobs (last 24h)

| # | job | workflow | fails | recovered | runs | fail rate | flaky? | scope | cause |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | `rampup` | test | 14 | 0 | 28 | 50% | flaky | main + PR | **timeout** — TestRampUp parent test exceeds ~300s job budget |
| 2 | `generate` | test | 10 | 0 | 28 | 36% | flaky | main + PR | **infra/CI** — generate job: platform changelog docs out of sync with codegen |
| 3 | `dashboard-arm` | build | 6 | 0 | 24 | 25% | flaky | PR | **product bug** — Docker build: TS2339 rows/pagination on timeout union in use-runs.tsx |
| 4 | `e2e-pgmq` | test | 5 | 1 | 28 | 18% | flaky | main + PR | **flaky test** — TestMultipleEvictionCycle intermittent failure in e2e-pgmq job |
| 5 | `e2e` | test | 5 | 1 | 28 | 18% | flaky | main + PR | **flaky test** — TestMultipleEvictionCycle intermittent failure in e2e job |
| 6 | `authdisabled` | build | 4 | 2 | 24 | 17% | flaky | PR | **product bug** — Docker build: TS2339 rows/pagination on timeout union in use-runs.tsx |
| 7 | `lite-amd` | build | 4 | 1 | 24 | 17% | flaky | PR | **product bug** — Docker build: TS2339 rows/pagination on timeout union in use-runs.tsx |
| 8 | `lint` | lint all | 5 | 0 | 33 | 15% | flaky | PR | **infra/CI** — pre-commit sync-typescript-changelog modified docs without commit |
| 9 | `cypress` | frontend / app | 4 | 0 | 21 | 19% | flaky | PR | **flaky test** — Cypress E2E: tenant selector timing and unhandled promise rejections |
| 10 | `frontend` | build | 4 | 0 | 24 | 17% | flaky | PR | **infra/CI** — frontend install: broken/duplicated key in pnpm-lock.yaml |

## Top 10 failing tests (last 24h)

| # | test | job | fails | runs | fail rate | flaky? | scope | cause |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | `TestRampUp` | `rampup` | 14 | 28 | 50% | flaky | main + PR | **timeout** — TestRampUp parent test exceeds ~300s job budget |
| 2 | `TestRampUp/normal_test` | `rampup` | 14 | 28 | 50% | flaky | main + PR | **timeout** — TestRampUp/normal_test exceeds ~300s job budget |
| 3 | `(unparsed)` | `generate` | 10 | 28 | 36% | flaky | main + PR | **infra/CI** — generate job: platform changelog docs out of sync with codegen |
| 4 | `(unparsed)` | `dashboard-arm` | 6 | 24 | 25% | flaky | PR | **product bug** — Docker build: TS2339 rows/pagination on timeout union in use-runs.tsx |
| 5 | `(unparsed)` | `authdisabled` | 5 | 24 | 21% | flaky | PR | **product bug** — Docker build: TS2339 rows/pagination on timeout union in use-runs.tsx |
| 6 | `(unparsed)` | `lint` | 5 | 33 | 15% | flaky | PR | **infra/CI** — pre-commit sync-typescript-changelog modified docs without commit |
| 7 | `(unparsed)` | `dashboard-amd` | 4 | 24 | 17% | flaky | PR | **product bug** — Docker build: TS2339 rows/pagination on timeout union in use-runs.tsx |
| 8 | `TestMultipleEvictionCycle` | `e2e` | 4 | 28 | 14% | flaky | main + PR | **flaky test** — TestMultipleEvictionCycle intermittent failure in e2e job |
| 9 | `TestMultipleEvictionCycle` | `e2e-pgmq` | 4 | 28 | 14% | flaky | main + PR | **flaky test** — TestMultipleEvictionCycle intermittent failure in e2e-pgmq job |
| 10 | `(unparsed)` | `lint` | 3 | 18 | 17% | flaky | PR | **infra/CI** — docs lint: prettier drift on typescript.mdx changelog |

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
