#!/bin/sh

# Prints the CHANGELOG.md section for the given tag. Prints nothing when the
# top section is for a different version, so CLI-only tags release without one.

TAG="${1#v}"

awk -v ver="$TAG" '
  /^## \[/ {
    if (seen++) exit
    if ($0 !~ "^## \\[" ver "\\]") exit
    next
  }
  seen { print }
' "CHANGELOG.md"
