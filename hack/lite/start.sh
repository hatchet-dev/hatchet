#!/bin/bash

# Run migration script
./hatchet-migrate
if [ $? -ne 0 ]; then
  echo "Migration script failed. Exiting..."
  exit 1
fi

# Generate config files
./hatchet-admin quickstart --skip certs --generated-config-dir ./config --overwrite=false

# Run the Go binary
./hatchet-lite --config ./config
