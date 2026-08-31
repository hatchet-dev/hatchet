#!/usr/bin/env bash
# Regenerates the TypeScript SDK reference docs into
# frontend/docs/content/docs/reference/typescript/.
#
# CI entry point: invoked as `task generate-sdk-docs-typescript` (runs from
# sdks/typescript). Requires only node + pnpm on the runner.
set -euo pipefail

cd "$(dirname "$0")/.."

pnpm install --frozen-lockfile
pnpm generate-docs
