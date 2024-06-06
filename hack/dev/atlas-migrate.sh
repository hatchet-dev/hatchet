#!/bin/bash

# check if the first argument is empty
if [ -z "$1" ]; then
  echo "Usage: $0 <version>"
  exit 1
fi

atlas migrate hash --dir "file://sql/migrations"

atlas migrate diff $1 \
  --dir "file://sql/migrations" \
  --to "file://sql/schema/schema.sql" \
  --dev-url "docker://postgres/15/dev?search_path=public"
