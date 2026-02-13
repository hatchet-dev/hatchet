#!/bin/bash

set -euo pipefail

get_token() {
  go run ./cmd/hatchet-admin token create --name local --tenant-id 707d0855-80ab-4e1f-a156-f1c4546cbf52 2>/dev/null
}

cat <<EOF
HATCHET_CLIENT_LOG_LEVEL=warn
HATCHET_CLIENT_TENANT_ID=707d0855-80ab-4e1f-a156-f1c4546cbf52
HATCHET_CLIENT_TLS_STRATEGY=none
HATCHET_CLIENT_TOKEN="$(get_token)"
EOF
