#!/bin/bash

set -eo pipefail

unset VIRTUAL_ENV

echo "Linting with ruff"
poetry run ruff check . --fix

echo "Formatting with black"
poetry run black . --color

echo "Type checking with mypy"
poetry run mypy --config-file=pyproject.toml

echo "Linting documentation with pydoclint"
poetry run pydoclint . --config pyproject.toml
