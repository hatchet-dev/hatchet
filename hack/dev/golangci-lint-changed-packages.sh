#!/usr/bin/env bash
# Run golangci-lint on only the Go packages that contain the given files.
#
# Used by the golangci-lint pre-commit hook instead of linting ./... on every
# commit. --new-from-rev HEAD limits reported issues to changed lines, so
# packages without changed files can't contribute findings.
#
# Directories where build constraints exclude every file (e.g. e2e-tagged
# packages) are skipped, matching how ./... treats them.
set -euo pipefail

if [ "$#" -eq 0 ]; then
  exit 0
fi

dirs=$(for f in "$@"; do dirname "./$f"; done | sort -u)

# shellcheck disable=SC2086 # word splitting is intended; repo paths contain no spaces
pkgs=$(go list -e -f '{{if or .GoFiles .CgoFiles .TestGoFiles .XTestGoFiles}}{{.Dir}}{{end}}' $dirs)

if [ -z "$pkgs" ]; then
  exit 0
fi

# shellcheck disable=SC2086
exec golangci-lint run --new-from-rev HEAD --fix --config=.golangci.yml --allow-parallel-runners $pkgs
