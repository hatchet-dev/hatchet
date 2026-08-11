#!/usr/bin/env bash
##
## CI entry point for generating the Python SDK reference docs
## (frontend/docs/content/docs/reference/python). Run from sdks/python,
## e.g. via `task generate-sdk-docs-python`.
##
## Requires python3.13 and poetry. Pages whose mkdocs export is unchanged
## (per docs/generator/manifest.json) are skipped. OPENAI_API_KEY is only
## needed when stale pages exist — the generator validates this itself and
## fails with a clear error, so seed/no-change runs are keyless.

set -euo pipefail

cd "$(dirname "$0")/.."

poetry install --extras docs --no-interaction

poetry run python -m docs.generator.generate "$@"
