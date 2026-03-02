#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

TEST_EXIT=0

cleanup() {
  set +e
  kill -9 $(jobs -p) 2>/dev/null
  wait 2>/dev/null
  pkill -9 -f hatchet-api
  pkill -9 -f hatchet-engine
  exit "$TEST_EXIT"
}
trap cleanup EXIT

task start-db
task write-e2e-env
task generate-local-encryption-keys
task migrate
task seed-cypress

bash ./hack/ci/start-api.sh &
bash ./hack/ci/start-engine.sh &

bash ./hack/ci/wait-for-http.sh http://127.0.0.1:8733/ready 120

cd frontend/app
pnpm run dev -- --host app.localtest.me --port 5173 &
bash ../../hack/ci/wait-for-http.sh http://app.localtest.me:5173 120

set +e
CYPRESS_BASE_URL=http://app.localtest.me:5173 pnpm run e2e:run
TEST_EXIT=$?
set -e
