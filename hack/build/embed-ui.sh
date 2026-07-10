#!/usr/bin/env bash
# Build the frontend dashboard and copy the compiled bundle into the CLI's
# embedded assets directory so it ships inside the `hatchet` binary.
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

# Drop source maps to keep the embedded bundle (and the binary) small.
find "$DEST" -name '*.map' -delete

# Keep the placeholder marker so `go:embed all:assets` still compiles from a
# clean checkout after the built assets are removed.
touch "$DEST/.gitkeep"

echo "Embedded UI bundle ready."
