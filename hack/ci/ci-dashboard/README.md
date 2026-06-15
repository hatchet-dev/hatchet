# CI Health Dashboard

A deterministic-first tool that mines GitHub Actions data into a pinned
[CI Health Dashboard](https://github.com/hatchet-dev/hatchet/issues) issue:
failure trends, the top failing jobs and tests, a flaky-vs-deterministic and
PR-vs-main classification, an LLM-labelled likely cause, and a "recent wins"
section of `ci-health` PRs.

Scripts do all the measuring. The agent only labels the *cause* of failure
signatures it has not seen before — that agent workflow is the
`ci-health-dashboard` skill (`.cursor/skills/ci-health-dashboard/SKILL.md`); this
README documents the tool the skill drives. State is cached locally so reruns are
cheap; `rm -rf .cache/` regenerates everything.

## Requirements

- [`uv`](https://docs.astral.sh/uv/) (each script is a self-contained PEP 723 script)
- `gh` authenticated with `repo` scope (`gh auth status`)

## Pipeline

| Stage | Script | LLM? | What it does |
| --- | --- | --- | --- |
| 1 | `collect.py` | no | Incrementally fetch runs + per-attempt jobs into `.cache/` |
| 2 | `parse_logs.py` | no | Download failed-step logs once; extract failing tests + signatures |
| 3 | `aggregate.py` | no | Top-10 jobs/tests, flaky/deterministic, PR/main, daily trend -> `out/analysis.json` |
| 4 | `wins.py` | no | Last 5 merged / open `ci-health` PRs -> `out/analysis.json` |
| - | `classify.py` | **yes** | Agent labels the cause of *new* signatures (cached by signature) |
| 5 | `render.py` | no | Build `out/issue.md` (tables + Mermaid trend) |
| 6 | `publish.py` | no | Find-or-create + update + pin the dashboard issue (`--publish` to apply) |

## Modes

- **Local** (default): generates `out/issue.md` and never touches GitHub state.
- **Publish**: creates the dashboard issue on first run, updates the same issue
  thereafter (find-or-create by hidden marker), and pins it.

## Usage

```bash
# LOCAL MODE: deterministic stages + render -> out/issue.md (no GitHub writes)
bash run.sh

# classify any new failure signatures (the agent step; see the skill)
uv run classify.py pending            # JSON of unclassified signatures + sample errors
uv run classify.py set --hash <h> --category "<category>" --reason "<one line>"

# re-render with the new classifications
uv run render.py

# PUBLISH MODE: create-or-update + pin the issue on hatchet-dev/hatchet
bash run.sh --publish                 # full pipeline + publish
# or just the publish step on an already-rendered out/issue.md:
uv run publish.py --publish
```

`classify.py` is the interface; deciding the category and reason is the agent
step, defined by the `ci-health-dashboard` skill.

## Cache

`.cache/` (gitignored, regenerable):

- `runs/<run_id>.json` — run metadata + per-attempt jobs (gating workflows)
- `job-failures/<job_id>.json` — parsed failing tests for one failed job
- `classifications.json` — signature -> {category, reason} (append-only)
- `meta.json` — last-collect timestamp + window

Completed runs/jobs/logs are immutable, so cached entries are never re-fetched.

A failure **signature** is
`workflow / job(matrix-stripped) / failing-step / normalized-error-line`
(digits, UUIDs, durations, ports masked). It is the dedup key for both trend
counting and classification.

## Scheduled Cursor Automation (suggested)

Create via the Automations editor (Agents Window):

- **Trigger:** cron, daily (e.g. `0 7 * * *`)
- **Repo / branch:** `hatchet-dev/hatchet` / the dashboard branch
- **Prompt:**

  > Use the `ci-health-dashboard` skill to refresh and publish the dashboard:
  > run the deterministic pipeline, classify any new failure signatures, render,
  > and publish (publish mode).

The agent workflow itself is defined as a project skill at
`.cursor/skills/ci-health-dashboard/SKILL.md`. Note the automation's git checkout
can only run scripts committed to the branch, so push the branch before relying on
the schedule.

## Labeling wins

Apply the `ci-health` label to PRs that fix CI/test flakiness so they surface in
the dashboard's wins section.
