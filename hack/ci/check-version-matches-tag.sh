#!/bin/sh
# Fails if the canonical pkg/version constant does not match the release tag.
#
# The engine reports pkg/version.Version to SDK clients, and library consumers such as
# github.com/hatchet-dev/hatchet/hatchetembed read it directly (no ldflag), so it must equal the tag
# being released. Run this from the release pipeline with the tag as the first argument, e.g.
#
#   ./hack/ci/check-version-matches-tag.sh v0.83.4
#
set -eu

TAG="${1:-}"
if [ -z "$TAG" ]; then
  echo "usage: $0 <tag>" >&2
  exit 2
fi

# Extract the string literal from `const Version = "..."` in pkg/version/version.go.
CONST=$(sed -n 's/^const Version = "\(.*\)".*/\1/p' pkg/version/version.go)

if [ -z "$CONST" ]; then
  echo "could not read Version from pkg/version/version.go" >&2
  exit 1
fi

if [ "$CONST" != "$TAG" ]; then
  echo "version mismatch: pkg/version.Version is '$CONST' but the tag is '$TAG'." >&2
  echo "Update pkg/version/version.go to '$TAG' before releasing." >&2
  exit 1
fi

echo "version OK: pkg/version.Version matches tag $TAG"
