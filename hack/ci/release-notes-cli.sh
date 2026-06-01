#!/bin/sh

TAG="${1#v}"

awk -v ver="$TAG" '
  /^## \[/ {
    if (seen++) exit
    if ($0 !~ "^## \\[" ver "\\]") exit 1
    next
  }
  seen { print }
' "CHANGELOG.md"
