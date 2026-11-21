#!/bin/bash

set -eo pipefail

echo "\nLinting with ruff"
poetry run ruff check . --fix

echo "Formatting with black"
poetry run black . --color

echo "\nType checking with mypy"
poetry run mypy --config-file=pyproject.toml

echo "\nType checking with pyright"
poetry run pyright

echo "\nType checking with ty"
poetry run ty check

echo "\nLinting documentation with pydoclint"
poetry run pydoclint . --config pyproject.toml
