#!/bin/bash

# Build DATABASE_URL from DATABASE_POSTGRES_* vars if unset
if [ -z "${DATABASE_URL:-}" ]; then
  export DATABASE_URL="postgresql://${DATABASE_POSTGRES_USERNAME:-hatchet}:${DATABASE_POSTGRES_PASSWORD:-hatchet}@${DATABASE_POSTGRES_HOST:-127.0.0.1}:${DATABASE_POSTGRES_PORT:-5431}/${DATABASE_POSTGRES_DB_NAME:-hatchet}?sslmode=${DATABASE_POSTGRES_SSL_MODE:-disable}"
fi

# Run migration script
./hatchet-migrate
if [ $? -ne 0 ]; then
  echo "Migration script failed. Exiting..."
  exit 1
fi

# Generate config files
./hatchet-admin quickstart --skip certs --generated-config-dir ./config --overwrite=false

# In authdisabled builds, mint and surface the default worker token
if ./hatchet-admin authdisabled; then
  if [ ! -s ./config/authdisabled-token ]; then
    ./hatchet-admin token create --config ./config --name authdisabled-default > ./config/authdisabled-token
  fi
  echo "================ authdisabled build: worker API token ==============="
  echo "Set HATCHET_CLIENT_TOKEN to the value below (also at ./config/authdisabled-token):"
  cat ./config/authdisabled-token
  echo
  echo "====================================================================="
fi

# Run the Go binary
exec ./hatchet-lite --config ./config
