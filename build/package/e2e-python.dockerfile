# Base Python environment
# -----------------------
FROM python:3.13-slim AS deployment

WORKDIR /hatchet/sdks/python

RUN pip install --no-cache-dir poetry==2.3.0

COPY sdks/python/ .

RUN poetry install --no-interaction --all-extras

CMD ["poetry", "run", "pytest", "-s", "-vvv", "--maxfail=5", "--capture=no", "--retries", "3", "--retry-delay", "2"]
