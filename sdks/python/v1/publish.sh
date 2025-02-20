#!/bin/bash
# This scripts generates and publishes the python package. 

# env name is required
if [ -z "$POETRY_PYPI_TOKEN_PYPI" ]; then
    echo "Please set POETRY_PYPI_TOKEN_PYPI variable"
    exit 1
fi

poetry build
poetry publish
