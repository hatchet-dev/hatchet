#!/bin/bash

set -eux

caddy start

set -a
. .env
set +a

npx --yes nodemon --signal SIGINT --config nodemon.api.json --exec go run ./cmd/hatchet-api
