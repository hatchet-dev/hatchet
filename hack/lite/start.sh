#!/bin/bash

# Start RabbitMQ
rabbitmq-server &

# Wait for RabbitMQ to be ready
until rabbitmqctl status; do
  echo "Waiting for RabbitMQ to start..."
  sleep 2
done

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
