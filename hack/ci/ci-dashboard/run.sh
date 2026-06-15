#!/usr/bin/env bash
# Deterministic pipeline for the CI health dashboard. Runs every stage that needs
# no LLM, then renders the issue body from whatever classifications already exist.
#
# Two modes:
#   bash run.sh            # LOCAL: collect -> parse -> aggregate -> wins -> render.
#                          #   Writes out/issue.md only; never touches GitHub state.
#   bash run.sh --publish  # PUBLISH: ...then create-or-update + pin the issue on
#                          #   hatchet-dev/hatchet.
#
# Classification of *new* failure signatures is the agent step (classify.py),
# done between render and publish (see README.md).
set -euo pipefail

cd "$(dirname "$0")"

uv run collect.py
uv run parse_logs.py
uv run aggregate.py
uv run wins.py

echo "--- pending classifications ---"
uv run classify.py stats

uv run render.py

if [[ "${1:-}" == "--publish" ]]; then
  uv run publish.py --publish
else
  echo "local mode: wrote out/issue.md (re-run with --publish to push to GitHub)"
fi
