#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
FRONTEND="$ROOT/frontend/app"
DEST="$ROOT/cmd/hatchet-cli/cli/internal/ui/assets"

if ! command -v python3 >/dev/null 2>&1; then
  echo "error: python3 (>= 3.10) is required to generate frontend/app/src/lib/generated" >&2
  exit 1
fi

echo "Generating docs snippets and examples"
cd "$ROOT/frontend/snippets"
python3 -X utf8 generate.py

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
