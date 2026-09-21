#!/usr/bin/env bash
# usage: EXPECT_FIPS=true|false hack/ci/assert-fips.sh IMAGE=/path/to/bin[,/path/to/bin...] ...
set -uo pipefail

EXPECT_FIPS="${EXPECT_FIPS:-true}"
GO_IMAGE="${GO_IMAGE:-golang:1.26-alpine}"
fail=0
tmp=$(mktemp -d)

cleanup() {
  rm -rf "$tmp"
  docker rm -f fipscheck >/dev/null 2>&1 || true
}
trap cleanup EXIT

[ $# -gt 0 ] || { echo "usage: $0 IMAGE=/path/to/bin[,/path/to/bin...] ..."; exit 2; }

for spec in "$@"; do
  image="${spec%%=*}"
  paths="${spec#*=}"
  docker rm -f fipscheck >/dev/null 2>&1 || true
  docker create --name fipscheck "$image" >/dev/null || { echo "::error::$image: could not create container"; fail=1; continue; }
  for p in ${paths//,/ }; do
    d=$(mktemp -d "$tmp/XXXXXX")
    docker cp "fipscheck:$p" "$d/bin" >/dev/null || { echo "::error::$image: $p missing from image"; fail=1; continue; }
    settings=$(docker run --rm -v "$d:/check:ro" "$GO_IMAGE" go version -m /check/bin 2>&1)
    if [ "$EXPECT_FIPS" = "true" ]; then
      if grep -q 'GOFIPS140=v1.0.0' <<<"$settings" && grep -Eq 'DefaultGODEBUG=.*fips140=on' <<<"$settings"; then
        echo "$image $p: validated FIPS module linked (GOFIPS140=v1.0.0, fips140=on)"
      else
        echo "::error::$image: $p is not a FIPS build"
        echo "$settings"
        fail=1
      fi
    elif grep -q 'GOFIPS140=' <<<"$settings"; then
      echo "::error::$image: $p links the FIPS module but this is not a -fips image"
      fail=1
    else
      echo "$image $p: not a FIPS build (expected)"
    fi
  done
done

exit $fail
