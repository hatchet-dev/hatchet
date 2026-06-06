#!/bin/bash

set -eux

set -a
. .env
set +a

npx --yes nodemon --signal SIGINT --config nodemon.engine.json --exec go run \
    -ldflags="-X main.Version=$(git rev-parse --short HEAD)" \
    ./cmd/hatchet-engine --no-graceful-shutdown
