#!/usr/bin/env bash
# Update the Go quickstart template's Hatchet SDK dependency.
#
# The CLI stores Go template files with .embed suffixes:
# - go.mod.embed
# - *.go.embed
#
# The quickstart renderer strips that suffix when generating a user project.
# This avoids two Go toolchain issues in the Hatchet repo:
# - a real go.mod under templates/go would create a nested module boundary that
#   //go:embed refuses to embed
# - real *.go files under the template directory could be treated as source
#   files in the main repo
#
# Dependabot cannot update go.mod.embed because its Go module file fetcher
# discovers Go modules by looking for go.mod and go.work.
#
# This script copies the template into a temp directory, strips .embed suffixes
# to simulate the generated project, runs go get and go mod tidy there, then
# copies the updated go.mod/go.sum back into the embedded template files.
#
# Usage:
#   bash hack/update-go-quickstart-deps.sh
#   bash hack/update-go-quickstart-deps.sh v0.88.0

set -euo pipefail

TEMPLATE_DIR="cmd/hatchet-cli/cli/templates/go"
VERSION="${1:-latest}"

if [ ! -f "${TEMPLATE_DIR}/go.mod.embed" ]; then
  echo "Error: ${TEMPLATE_DIR}/go.mod.embed not found. Run from the repo root." >&2
  exit 1
fi

TMPDIR="$(mktemp -d)"
trap 'rm -rf "${TMPDIR}"' EXIT

echo "Copying template to temp directory..."
cp -a "${TEMPLATE_DIR}/." "${TMPDIR}/"

echo "Stripping .embed suffixes..."
find "${TMPDIR}" -name '*.embed' -print0 | while IFS= read -r -d '' f; do
  mv "${f}" "${f%.embed}"
done

echo "Updating github.com/hatchet-dev/hatchet to ${VERSION}..."
(
  cd "${TMPDIR}"
  go get "github.com/hatchet-dev/hatchet@${VERSION}"
  go mod tidy
)

echo "Copying updated files back..."
cp "${TMPDIR}/go.mod" "${TEMPLATE_DIR}/go.mod.embed"
cp "${TMPDIR}/go.sum" "${TEMPLATE_DIR}/go.sum"

echo "Done. Updated files:"
echo "  ${TEMPLATE_DIR}/go.mod.embed"
echo "  ${TEMPLATE_DIR}/go.sum"
