## Hatchet Python Quickstart - {{ .Name }}

This is an example project demonstrating how to use Hatchet with Python. For detailed setup instructions, see the [Hatchet Setup Guide](https://docs.hatchet.run/home/setup).

## Prerequisites

Before running this project, make sure you have the following:

1. [Python v3.10 or higher](https://www.python.org/downloads/)
{{- if eq .PackageManager "poetry"}}
2. [Poetry](https://python-poetry.org/docs/#installation) for dependency management
{{- else if eq .PackageManager "uv"}}
2. [uv](https://docs.astral.sh/uv/) for dependency management
{{- else if eq .PackageManager "pip"}}
2. pip (included with Python)
{{- end}}

## Setup

1. Clone the repository:

```bash
git clone https://github.com/hatchet-dev/hatchet-python-quickstart.git
cd hatchet-python-quickstart
```

2. Set the required environment variable `HATCHET_CLIENT_TOKEN` created in the [Getting Started Guide](https://docs.hatchet.run/home/hatchet-cloud-quickstart).

```bash
export HATCHET_CLIENT_TOKEN=<token>
```

> Note: If you're self hosting you may need to set `HATCHET_CLIENT_TLS_STRATEGY=none` to disable TLS

3. Install the project dependencies:

```bash
{{- if eq .PackageManager "poetry"}}
poetry install
{{- else if eq .PackageManager "uv"}}
# Create a virtual environment (if it doesn't exist)
uv venv

# Install dependencies
uv pip install -e .
{{- else if eq .PackageManager "pip"}}
# Create a virtual environment
python -m venv .venv

# Activate the virtual environment
source .venv/bin/activate  # On Windows: .venv\Scripts\activate

# Install dependencies
pip install -r requirements.txt
{{- end}}
```

### Running an example

{{- if eq .PackageManager "pip"}}
> **Note**: Make sure your virtual environment is activated before running these commands:
> ```bash
> source .venv/bin/activate  # On Windows: .venv\Scripts\activate
> ```
{{- end}}

1. Start a Hatchet worker by running the following command:

```shell
{{- if eq .PackageManager "poetry"}}
poetry run python src/worker.py
{{- else if eq .PackageManager "uv"}}
uv run python src/worker.py
{{- else if eq .PackageManager "pip"}}
python src/worker.py
{{- end}}
```

2. To run the example workflow, open a new terminal and run the following command:

```shell
{{- if eq .PackageManager "poetry"}}
poetry run python src/run.py
{{- else if eq .PackageManager "uv"}}
uv run python src/run.py
{{- else if eq .PackageManager "pip"}}
# Make sure to activate the venv first: source .venv/bin/activate
python src/run.py
{{- end}}
```

This will trigger the workflow on the worker running in the first terminal and print the output to the the second terminal.

{{- if eq .PackageManager "uv"}}

> **Note**: `uv run` automatically uses the virtual environment created by `uv venv`. You don't need to manually activate it.
{{- end}}
