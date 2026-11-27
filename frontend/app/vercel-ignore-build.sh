#!/usr/bin/env bash
# Vercel's ignoreCommand: exit 0 skips the Storybook build, exit 1 runs it.
set -uo pipefail

# The last successful deploy on this branch, so a push with several commits is
# compared as a whole. Empty on a branch's first deploy.
base="${VERCEL_GIT_PREVIOUS_SHA:-HEAD^}"

# A force-push can remove that commit, and git diff would then fail the deploy.
if ! git cat-file -e "${base}^{commit}" 2>/dev/null; then
  exit 1
fi

git diff --quiet "$base" HEAD -- .
