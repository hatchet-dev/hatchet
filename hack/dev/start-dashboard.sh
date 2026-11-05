#!/bin/bash

set -eux

set -a
. .env
set +a

npx --yes nodemon --signal SIGINT --config nodemon.api.json --exec go run ./cmd/hatchet-dashboard
