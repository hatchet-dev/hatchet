#!/bin/bash

set -eux

if command -v caddy >/dev/null 2>&1; then
    caddy start
fi

set -a
. .env
set +a

npx --yes nodemon --signal SIGINT --config nodemon.api.json --exec go run \
    -ldflags="-X main.Version=$(git rev-parse --short HEAD)" \
     ./cmd/hatchet-api
