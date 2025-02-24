echo "Formatting with black"
poetry run black . --color

echo "\nFormatting with isort"
poetry run isort .

echo "\nType checking with mypy"
poetry run mypy --config-file=pyproject.toml

echo "\nLinting with ruff"
poetry run ruff check . --fix