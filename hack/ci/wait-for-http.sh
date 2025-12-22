#!/usr/bin/env bash

set -euo pipefail

URL="${1:-}"
TIMEOUT_SECONDS="${2:-60}"

if [ -z "$URL" ]; then
  echo "usage: wait-for-http.sh <url> [timeoutSeconds]" >&2
  exit 2
fi

start_ts="$(date +%s)"

while true; do
  if curl -fsS "$URL" >/dev/null; then
    echo "ready: $URL"
    exit 0
  fi

now_ts="$(date +%s)"
elapsed="$((now_ts - start_ts))"
if [ "$elapsed" -ge "$TIMEOUT_SECONDS" ]; then
  echo "timeout after ${TIMEOUT_SECONDS}s waiting for $URL" >&2
  exit 1
fi

sleep 1
done
