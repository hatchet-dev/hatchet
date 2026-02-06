FROM python:3.11-slim

WORKDIR /app
# Install Poetry
RUN pip install poetry

# Copy dependency files
COPY pyproject.toml poetry.lock ./

# Install dependencies
RUN poetry config virtualenvs.create false && poetry install --no-root --only main

# Copy source code
COPY src ./src

CMD ["poetry", "run", "python", "src/worker.py"]
