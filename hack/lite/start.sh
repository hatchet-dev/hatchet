#!/bin/bash

# Run migration script
./hatchet-migrate
if [ $? -ne 0 ]; then
  echo "Migration script failed. Exiting..."
  exit 1
fi

# Generate config files (also generates the no-auth keyset when SERVER_AUTH_NO_AUTH_ENABLED is set)
./hatchet-admin quickstart --skip certs --generated-config-dir ./config --overwrite=false

# In no-auth mode, mint and surface the default worker token
if [ "$SERVER_AUTH_NO_AUTH_ENABLED" = "true" ] || [ "$SERVER_AUTH_NO_AUTH_ENABLED" = "t" ]; then
  if [ ! -s ./config/noauth-token ]; then
    ./hatchet-admin token create --config ./config --no-auth --name noauth-default > ./config/noauth-token
  fi
  echo "=================== no-auth mode: worker API token ==================="
  echo "Set HATCHET_CLIENT_TOKEN to the value below (also at ./config/noauth-token):"
  cat ./config/noauth-token
  echo
  echo "====================================================================="
fi

# Run the Go binary
./hatchet-lite --config ./config
