#!/bin/bash

# Check whether DATABASE_URL is set
if [ -z "$DATABASE_URL" ]; then
  echo "DATABASE_URL is not set"
  exit 1
fi

# Check for prisma migrations
MIGRATION_NAME=$(psql "$DATABASE_URL" -t -c "SELECT migration_name FROM _prisma_migrations ORDER BY started_at DESC LIMIT 1;" 2>/dev/null | xargs)

if [ $? -eq 0 ] && [ -n "$MIGRATION_NAME" ]; then
  echo "Using existing prisma migration: $MIGRATION_NAME"

  atlas migrate apply \
    --url "$DATABASE_URL" \
    --baseline "$MIGRATION_NAME" \
    --dir "file://sql/migrations"
else
  echo "No prisma migration found. Applying all migrations..."

  atlas migrate apply \
    --url "$DATABASE_URL" \
    --dir "file://sql/migrations"
fi

# if either of the above commands failed, exit with an error
if [ $? -ne 0 ]; then
  echo "Migration failed. Exiting..."
  exit 1
fi
