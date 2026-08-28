#!/usr/bin/env bash
# Generates the Ruby SDK reference docs (frontend/docs/content/docs/reference/ruby/)
# from the YARD docstrings in src/lib/ and the RBS signatures in src/sig/.
#
# Run from sdks/ruby/ (wired up as `task generate-sdk-docs-ruby` in the root Taskfile).
# Requires ruby >= 3.2. Installs yard if it is not already available.
set -euo pipefail

cd "$(dirname "$0")/.."

if ! ruby -e 'require "yard"' >/dev/null 2>&1; then
  gem install yard --no-document
fi

ruby docs/generate.rb
