#!/bin/bash

MIGRATION_NAME=$1

# check if the first argument is empty
if [ -z "$MIGRATION_NAME" ]; then
  MIGRATION_NAME="temp"
fi

atlas migrate hash --dir "file://sql/migrations"

# psql "CREATE DATABASE atlas IF NOT EXISTS" -U postgres
atlas migrate diff $MIGRATION_NAME \
  --dir "file://sql/migrations" \
  --to "file://sql/schema/schema.sql" \
  --dev-url "postgresql://hatchet:hatchet@127.0.0.1:5431/atlas?sslmode=disable&search_path=public"
