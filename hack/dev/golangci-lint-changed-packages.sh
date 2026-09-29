#!/usr/bin/env bash
# Pre-commit entrypoint for golangci-lint.
#
# Linting is scoped to the packages of the given files: --new-from-rev HEAD
# limits reported issues to changed lines, so other packages can't contribute
# findings and analyzing them only costs time.
#
# Directories where build constraints exclude every file (e.g. e2e-tagged
# packages) must be skipped; golangci-lint fails to typecheck them when they
# are passed explicitly.
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
