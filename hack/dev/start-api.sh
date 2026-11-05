#!/bin/bash

set -eux

caddy start

set -a
. .env
set +a

# the engine runs as a separate process on this host and serves its metrics on :9090
export SERVER_PROMETHEUS_ADDRESS=":9091"

npx --yes nodemon --signal SIGINT --config nodemon.api.json --exec go run \
    -ldflags="-X main.Version=$(git rev-parse --short HEAD)" \
     ./cmd/hatchet-api
