# !/bin/bash

set -eo pipefail

poetry run mkdocs build
cp -r site/* ../../frontend/docs/public/sdks/python
