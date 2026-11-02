#!/usr/bin/env bash
# Packs the TypeScript SDK and @hatchet-dev/serverless from their build output into ./vendor so
# the example depends on tarballs that travel with it: a Vercel deployment uploads this directory
# alone, where the repository's sibling packages do not exist. Run the builds first:
#   cd sdks/typescript && pnpm install && pnpm tsc:build && pnpm prepublish
#   cd ../typescript-serverless && pnpm install && pnpm build
set -euo pipefail

HERE="$(cd "$(dirname "$0")/.." && pwd)"
ROOT="$(cd "$HERE/../../.." && pwd)"
SDK_DIST="$ROOT/sdks/typescript/dist"
PKG="$ROOT/sdks/typescript-serverless"

[ -f "$SDK_DIST/package.json" ] || { echo "$SDK_DIST is not built; run pnpm tsc:build && pnpm prepublish in sdks/typescript" >&2; exit 1; }
[ -d "$PKG/dist" ] || { echo "$PKG/dist is not built; run pnpm build in sdks/typescript-serverless" >&2; exit 1; }

mkdir -p "$HERE/vendor"
(cd "$SDK_DIST" && pnpm pack --out "$HERE/vendor/hatchet-dev-typescript-sdk.tgz" >/dev/null)
(cd "$PKG" && pnpm pack --out "$HERE/vendor/hatchet-dev-serverless.tgz" >/dev/null)
ls -la "$HERE/vendor"/*.tgz
