FROM python:3.11-slim

WORKDIR /app

{{- if eq .PackageManager "poetry"}}
# Install Poetry
RUN pip install poetry

# Copy dependency files
COPY pyproject.toml poetry.lock ./

# Install dependencies
RUN poetry config virtualenvs.create false && poetry install --no-root --only main

# Copy source code
COPY src ./src

CMD ["poetry", "run", "python", "src/worker.py"]

{{- else if eq .PackageManager "uv"}}
# Install uv
RUN pip install uv

# Copy dependency files
COPY pyproject.toml ./

# Install dependencies
RUN uv venv && uv pip install -e .

# Copy source code
COPY src ./src

CMD ["uv", "run", "python", "src/worker.py"]

{{- else if eq .PackageManager "pip"}}
# Copy dependency files
COPY requirements.txt ./

# Install dependencies
RUN pip install --no-cache-dir -r requirements.txt

# Copy source code
COPY src ./src

CMD ["python", "src/worker.py"]
{{- end}}
