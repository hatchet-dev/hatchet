#!/bin/bash

set -a
. .env
set +a

nodemon --signal SIGINT --config nodemon.engine.json --exec go run ./cmd/hatchet-engine
