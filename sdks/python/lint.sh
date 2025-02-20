poetry run black . --color
poetry run isort .
poetry run mypy --config-file=pyproject.toml
poetry run ruff . --fix
