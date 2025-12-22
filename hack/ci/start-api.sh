#!/usr/bin/env bash

set -euo pipefail

# CI-friendly API startup (no caddy, no nodemon).

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

set -a
. .env
set +a

exec go run ./cmd/hatchet-api
