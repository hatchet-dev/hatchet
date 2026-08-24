#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
FRONTEND="$ROOT/frontend/app"
DEST="$ROOT/cmd/hatchet-cli/cli/internal/ui/assets"

echo "Building frontend in $FRONTEND"
cd "$FRONTEND"
pnpm install --frozen-lockfile
pnpm run build

echo "Copying bundle into $DEST"
rm -rf "$DEST"
mkdir -p "$DEST"
cp -R "$FRONTEND/dist/." "$DEST/"

find "$DEST" -name '*.map' -delete

touch "$DEST/.gitkeep"

echo "Embedded UI bundle ready."
