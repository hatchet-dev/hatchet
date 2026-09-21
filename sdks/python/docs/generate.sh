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

pv=$(poetry --version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)
case "$pv" in
  "" | 0.* | 1.* | 2.0.* | 2.1.*)
    echo "error: poetry >= 2.2 required (poetry.lock uses PEP 735 dependency groups), found ${pv:-none}" >&2
    exit 1
    ;;
esac

poetry install --extras docs --no-interaction

poetry run python -m docs.generator.generate "$@"
