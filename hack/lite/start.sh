#!/bin/bash

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
./hatchet-lite --config ./config
