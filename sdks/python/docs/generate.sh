#!/usr/bin/env bash
##
## CI entry point for generating the Python SDK reference docs
## (frontend/docs/content/docs/reference/python). Run from sdks/python,
## e.g. via `task generate-sdk-docs-python`.
##
## Requires python3.13 and poetry. The conversion is fully deterministic,
## no API keys or network access needed beyond dependency installation.

set -euo pipefail

cd "$(dirname "$0")/.."

poetry install --extras docs --no-interaction

poetry run python -m docs.generator.generate "$@"
