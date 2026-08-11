#!/usr/bin/env bash
##
## CI entry point for generating the Python SDK reference docs
## (frontend/docs/content/docs/reference/python). Run from sdks/python,
## e.g. via `task generate-sdk-docs-python`.
##
## Requires python3.13, poetry, and OPENAI_API_KEY. Pages whose mkdocs export
## is unchanged (per docs/generator/manifest.json) are skipped, so runs where
## nothing changed make no OpenAI calls.

set -euo pipefail

cd "$(dirname "$0")/.."

if [[ -z "${OPENAI_API_KEY:-}" || "${OPENAI_API_KEY:-}" == "fake-key" ]]; then
  echo "error: OPENAI_API_KEY is not set (or is a placeholder). Refusing to generate docs." >&2
  exit 1
fi

poetry install --extras docs --no-interaction

poetry run python -m docs.generator.generate "$@"
