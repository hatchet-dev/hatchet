---
name: ci-health-dashboard
description: >-
  Refresh the Hatchet CI Health Dashboard: run the pipeline in
  hack/ci/ci-dashboard, classify any NEW failure signatures (the agent step),
  and render out/issue.md locally. Publishes the dashboard issue only when
  explicitly asked. Invoke directly (e.g. from the scheduled CI-health
  automation).
disable-model-invocation: true
---

# CI Health Dashboard

This skill owns the **agent workflow**: run the pipeline, classify new failure
signatures, render, and (only on request) publish. The tool's mechanics —
pipeline stages, cache model, signatures, modes, requirements — live in
`hack/ci/ci-dashboard/README.md`; read it for anything not covered here.

Default to **local mode** (render `out/issue.md`, no GitHub writes). Publish only
when the request explicitly asks to publish.

All commands run from `hack/ci/ci-dashboard/`.

## Hard rules — do not improvise

The dashboard lives in exactly one issue, **#4204**, and is published by one
GitHub Actions workflow. If something blocks you, **stop and report** what failed;
do not work around it. Specifically, NEVER:

- create a GitHub issue (`gh issue create`) — the dashboard issue already exists;
- "probe" or test permissions by creating throwaway issues/PRs;
- open a workaround PR or add/modify workflows to get around a permission error;
- publish to any issue other than #4204 (unless the user explicitly says so).

A `Resource not accessible by integration (updateIssue)` error means the current
token lacks `issues: write` (expected for the Cursor cloud token). That is normal —
use the staging path below, which lets the GitHub Action do the write.

## Workflow

```
- [ ] 1. bash run.sh            # pipeline + render (local); prints pending count
- [ ] 2. classify new signatures (the agent step)  -- only if pending > 0
- [ ] 3. uv run render.py       # fold the new causes into out/issue.md
- [ ] 4. publish -- ONLY if explicitly asked (see Publishing)
```

## The agent step: classification

`run.sh` prints how many signatures need a cause. If pending is 0, skip to render.

```bash
uv run classify.py pending   # JSON: each unclassified signature + a sample error
```

For each item, choose one category from the signature
(`workflow / job / step / error`) and the `sample_error`, then record it:

```bash
uv run classify.py set --hash <sig_hash> --category "<category>" --reason "<one line>"
```

| Category | Pick when |
| --- | --- |
| `timeout` | Exceeded a time budget — duration near a known cap (~300s, 340s), "context deadline exceeded", "timed out". |
| `infra/CI` | CI environment, not the product: service not ready ("connection refused" to a localhost port), failing setup step, codegen "check for diff" drift, missing CI deps. |
| `product bug` | A real engine/SDK defect causes the failure (name the subsystem). Often closed by a linked fix PR. |
| `flaky test` | Test is non-deterministic (timing/ordering/race in the test) but the product is fine. |
| `dependency` | A third-party / Dependabot bump broke the build. |
| `data/env` | Bad fixtures, env vars, DB/migration state. |
| `unknown` | Sample is noise (a `set -x` trace, a git fetch line) and job/step context is insufficient. Prefer this over guessing. |

When a high-fail-count signature has a noisy `sample_error`, pull one failing run
for context before labelling — `gh run list --repo hatchet-dev/hatchet --workflow
<workflow> --status failure`, then `gh run view --log-failed <run-id>`. Keep it
targeted; the cache already holds the logs.

Finish with `uv run classify.py stats` (expect `"pending": 0`), then re-render.

## Publishing

Skip unless the request explicitly says to publish. The dashboard is always
**[#4204](https://github.com/hatchet-dev/hatchet/issues/4204)** (`config.DASHBOARD_ISSUE`).
There are two paths; the issue write itself is done by `publish.py`, which edits
#4204 in place and pins it (never creates an issue).

### Default path: stage + let CI publish (works without `issues: write`)

This is the path for the Cursor automation / cloud agent, whose token cannot edit
issues. Stage the rendered body and push it; the `Publish CI Health Dashboard`
workflow (`.github/workflows/ci-health-dashboard-publish.yml`, which has
`issues: write`) updates #4204 on push.

```bash
bash run.sh --stage   # render + copy out/issue.md -> staging/issue.md
git add staging/issue.md && git commit -m "chore(ci): refresh CI health dashboard" \
  && git push origin HEAD:ci-health-dashboard
```

Then confirm the workflow run succeeded:
`gh run list --repo hatchet-dev/hatchet --workflow ci-health-dashboard-publish.yml --limit 1`.
If a cloud runtime can only open a PR (not push to the branch), say so and stop —
publishing happens when that PR merges to `ci-health-dashboard`.

### Direct path: only when your `gh` has `issues: write` (e.g. local, as yourself)

```bash
uv run publish.py            # DRY RUN — confirm it says "update issue #4204"
uv run publish.py --publish  # edit + pin #4204 directly
```

Use `--issue <n>` only if the user explicitly asks for a different issue.

## Wins label

When a PR fixes a CI/test flake, label it so it shows up in the dashboard's wins
section: `gh pr edit <number> --repo hatchet-dev/hatchet --add-label ci-health`.
