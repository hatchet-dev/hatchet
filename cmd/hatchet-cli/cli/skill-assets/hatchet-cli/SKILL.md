---
name: hatchet-cli
description: Hatchet CLI skills for managing workflows, workers, and runs. Use this skill when triggering workflows, starting workers, debugging runs, replaying failed runs, or setting up the Hatchet CLI and profiles.
license: MIT
metadata:
  author: Hatchet
  version: "1.0.0"
  organization: Hatchet
  date: February 2026
  abstract: Skills for using the Hatchet CLI to manage workflows, workers, and runs. Includes instructions for triggering workflows, starting workers, debugging runs, replaying runs, and setting up the CLI and profiles. Each skill references detailed documentation with step-by-step instructions and best practices.
---

# Hatchet CLI Agent Skills

This skill package teaches AI agents how to use the Hatchet CLI to manage workflows, workers, and runs.

## When to use these skills

Read the relevant reference document before performing any Hatchet CLI task:

- **Setting up the CLI or creating a profile** → `references/setup-cli.md`
- **Starting a worker** → `references/start-worker.md`
- **Triggering a workflow and waiting for results** → `references/trigger-and-watch.md`
- **Debugging a failed or stuck run** → `references/debug-run.md`
- **Replaying a run with the same or new input** → `references/replay-run.md`

## Key conventions

- Always specify a profile with `-p HATCHET_PROFILE` unless a default profile is set.
- Use `-o json` for machine-readable output when parsing responses.
- Write workflow input to a temp file (e.g. `/tmp/hatchet-input-$(date +%s)-$$.json`) to avoid collisions.
- Clean up temp files after use.
