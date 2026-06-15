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

## Workflow

```
- [ ] 1. bash run.sh            # pipeline + render (local); prints pending count
- [ ] 2. classify new signatures (the agent step)  -- only if pending > 0
- [ ] 3. uv run render.py       # fold the new causes into out/issue.md
- [ ] 4. uv run publish.py --publish   -- ONLY if explicitly asked to publish
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

Skip unless the request explicitly says to publish. When it does:

```bash
uv run publish.py            # DRY RUN first — prints create vs update
uv run publish.py --publish  # create first time, update + pin the same issue after
```

The canonical dashboard issue is **[#4204](https://github.com/hatchet-dev/hatchet/issues/4204)**.
`publish.py` finds it by the hidden `<!-- ci-health-dashboard:v1 -->` marker in the
issue body, so it should resolve to #4204 automatically. Before `--publish`, confirm
the dry run says `update issue #4204` — if it says "create a new issue" or a different
number, stop and investigate (the marker was dropped from #4204, or a second open issue
picked it up) instead of creating a duplicate.

## Wins label

When a PR fixes a CI/test flake, label it so it shows up in the dashboard's wins
section: `gh pr edit <number> --repo hatchet-dev/hatchet --add-label ci-health`.
