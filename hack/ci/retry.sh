#!/bin/sh
# Run a command up to 3 times with backoff (10s, then 30s) so that transient
# network errors during dependency installs (registry timeouts, dropped TLS
# handshakes on postinstall downloads, go proxy blips) do not fail the job.
#
# Trade-off: a retry does not fix a deterministic failure, it only repeats it,
# and on a bad day it adds at most two more install attempts plus 40s of sleep
# before the job fails. That is much cheaper than re-running the job by hand.
#
# Usage: hack/ci/retry.sh <command> [args...]
# Exits with the exit code of the last attempt.

set -u

if [ "$#" -eq 0 ]; then
  echo "usage: retry.sh <command> [args...]" >&2
  exit 2
fi

attempt=1
max_attempts=3
status=0

while :; do
  "$@" && exit 0
  status=$?

  if [ "$attempt" -ge "$max_attempts" ]; then
    echo "retry: attempt $attempt/$max_attempts failed with exit code $status, giving up: $*" >&2
    exit "$status"
  fi

  if [ "$attempt" -eq 1 ]; then
    delay=10
  else
    delay=30
  fi

  echo "retry: attempt $attempt/$max_attempts failed with exit code $status, retrying in ${delay}s: $*" >&2
  sleep "$delay"
  attempt=$((attempt + 1))
done
