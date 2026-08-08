#!/usr/bin/env bash
# Deterministic pipeline for the CI health dashboard. Runs every stage that needs
# no LLM, then renders the issue body from whatever classifications already exist.
#
# Modes:
#   bash run.sh            # LOCAL: collect -> parse -> aggregate -> wins -> render.
#                          #   Writes out/issue.md only; never touches GitHub state.
#   bash run.sh --stage    # ...then copy out/issue.md -> staging/issue.md so it can
#                          #   be committed; the publish GitHub Action (issues:write)
#                          #   updates the dashboard issue on push of that file.
#   bash run.sh --publish  # ...then update + pin the issue directly (needs a gh
#                          #   token with issues:write, e.g. running locally as you).
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

case "${1:-}" in
  --stage)
    mkdir -p staging
    cp out/issue.md staging/issue.md
    echo "staged: staging/issue.md (commit + push to trigger the publish workflow)"
    ;;
  --publish)
    uv run publish.py --publish
    ;;
  *)
    echo "local mode: wrote out/issue.md (--stage to commit a body for CI publish, --publish to push directly)"
    ;;
esac
