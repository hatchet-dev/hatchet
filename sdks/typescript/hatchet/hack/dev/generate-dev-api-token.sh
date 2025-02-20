#!/bin/bash
# This scripts generates a local API token.

set -eux

set -a
. .env
set +a

go run ./cmd/hatchet-admin token create --name "local" --tenant-id 707d0855-80ab-4e1f-a156-f1c4546cbf52
