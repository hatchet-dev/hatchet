#!/usr/bin/env bash
# Pre-commit entrypoint for golangci-lint.
#
# golangci-lint is built and run with the Go toolchain go.mod selects, which is
# the one CI uses. Under a newer go, golangci-lint fails to typecheck because it
# can't read that release's export data, and the go on PATH may be newer.
#
# Linting is scoped to the packages of the given files: --new-from-rev HEAD
# limits reported issues to changed lines, so other packages can't contribute
# findings and analyzing them only costs time.
#
# Directories where build constraints exclude every file (e.g. e2e-tagged
# packages) must be skipped; golangci-lint fails to typecheck them when they
# are passed explicitly.
set -euo pipefail

golangci_lint_version=v2.12.2

if [ "$#" -eq 0 ]; then
  exit 0
fi

GOTOOLCHAIN=$(awk '$1 == "go" { v = "go" $2 } $1 == "toolchain" { t = $2 } END { print t ? t : v }' go.mod)
export GOTOOLCHAIN

dirs=$(for f in "$@"; do dirname "./$f"; done | sort -u)

# shellcheck disable=SC2086 # word splitting is intended; repo paths contain no spaces
pkgs=$(go list -e -f '{{if or .GoFiles .CgoFiles .TestGoFiles .XTestGoFiles}}{{.Dir}}{{end}}' $dirs)

if [ -z "$pkgs" ]; then
  exit 0
fi

# Built once per version and toolchain, next to pre-commit's environments, not
# with go run: go run queries the module proxy on every call, so commits would
# need the network.
cache_dir=${PRE_COMMIT_HOME:-${XDG_CACHE_HOME:-$HOME/.cache}/pre-commit}
bin_dir=$cache_dir/golangci-lint-$golangci_lint_version-$GOTOOLCHAIN
if [ ! -x "$bin_dir/golangci-lint" ]; then
  GOBIN=$bin_dir go install "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$golangci_lint_version"
fi

# shellcheck disable=SC2086
exec "$bin_dir/golangci-lint" run --new-from-rev HEAD --fix --config=.golangci.yml --allow-parallel-runners $pkgs
