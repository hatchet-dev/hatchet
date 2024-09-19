#!/bin/bash

MIGRATION_NAME=$1

# check if the first argument is empty
if [ -z "$MIGRATION_NAME" ]; then
  MIGRATION_NAME="temp"
fi

atlas migrate hash --dir "file://sql/migrations"

atlas migrate diff $MIGRATION_NAME \
  --dir "file://sql/migrations" \
  --to "file://sql/schema/schema.sql" \
  --dev-url "docker://postgres/15/dev?search_path=public"
