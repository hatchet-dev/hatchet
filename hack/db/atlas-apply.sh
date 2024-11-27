#!/bin/bash

# Check whether DATABASE_URL is set
if [ -z "$DATABASE_URL" ]; then
  echo "DATABASE_URL is not set"
  exit 1
fi

# Wait up to 30 seconds for the database to be ready
echo "Waiting for database to be ready..."
timeout 30s bash -c '
until psql "$DATABASE_URL" -c "\q" 2>/dev/null; do
  sleep 1
done
'
if [ $? -eq 124 ]; then
  echo "Timed out waiting for the database to be ready"
  exit 1
fi

# Function to attempt a psql connection with the given DATABASE_URL
attempt_psql_connection() {
  local db_url=$1
  psql "$db_url" -t -c "SELECT 1;" 2>/dev/null
  return $?
}

# Check if sslmode is set in the DATABASE_URL
if [[ ! "$DATABASE_URL" =~ sslmode ]]; then
  # Attempt a secure psql connection first if sslmode is not set
  SECURE_DB_URL="${DATABASE_URL}?sslmode=require"
  attempt_psql_connection "$SECURE_DB_URL"
  if [ $? -ne 0 ]; then
    # If secure connection fails, use sslmode=disable
    echo "Secure connection failed. Using sslmode=disable"

    DATABASE_URL="${DATABASE_URL}?sslmode=disable"
  else
    DATABASE_URL="$SECURE_DB_URL"
  fi
fi

echo "DATABASE_URL: $DATABASE_URL"

echo "Applying migrations via atlas..."

atlas migrate apply \
  --url "$DATABASE_URL" \
  --dir "file://sql/migrations"


# if either of the above commands failed, exit with an error
if [ $? -ne 0 ]; then
  echo "Migration failed. Exiting..."
  exit 1
fi
