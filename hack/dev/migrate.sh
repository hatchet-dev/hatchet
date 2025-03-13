#!/bin/bash

set -eux
set -a

echo "Working dir"
ls -al

. .env
set +a

go run ./cmd/hatchet-migrate
