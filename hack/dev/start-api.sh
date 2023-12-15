#!/bin/bash

caddy start

set -a
. .env
set +a

nodemon --signal SIGINT --config nodemon.api.json --exec go run ./cmd/hatchet-api
