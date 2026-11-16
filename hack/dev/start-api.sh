#!/bin/bash

set -eux

caddy start

set -a
. .env
set +a

# the engine serves its metrics on :9090 and the telemetry stack's Prometheus publishes :9091
export SERVER_PROMETHEUS_ADDRESS=":9092"

npx --yes nodemon --signal SIGINT --config nodemon.api.json --exec go run \
    -ldflags="-X main.Version=$(git rev-parse --short HEAD)" \
     ./cmd/hatchet-api
