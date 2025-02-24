#!/bin/bash

set -eux
set -a
. .env
set +a

go run ./cmd/hatchet-migrate
