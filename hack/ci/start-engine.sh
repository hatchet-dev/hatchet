#!/usr/bin/env bash

set -euo pipefail

# CI-friendly engine startup (no nodemon).

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

set -a
. .env
set +a

exec go run ./cmd/hatchet-engine --no-graceful-shutdown
