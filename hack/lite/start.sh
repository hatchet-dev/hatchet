#!/bin/bash

rabbitmq-server &

# Wait up to 60 seconds for RabbitMQ to be ready
echo "Waiting for RabbitMQ to be ready..."

timeout 60s bash -c '
until rabbitmqctl status; do
  sleep 2
  echo "Waiting for RabbitMQ to start..."
done
'

if [ $? -eq 124 ]; then
  echo "Timed out waiting for the database to be ready"
  exit 1
fi

# Run migration script
./atlas-apply.sh
if [ $? -ne 0 ]; then
  echo "Migration script failed. Exiting..."
  exit 1
fi

# Generate config files
./hatchet-admin quickstart --skip certs --generated-config-dir ./config --overwrite=false

# Run the Go binary
./hatchet-lite --config ./config
