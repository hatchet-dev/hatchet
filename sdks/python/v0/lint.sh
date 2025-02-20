poetry run black . --color
poetry run isort .
poetry run mypy --config-file=pyproject.toml
