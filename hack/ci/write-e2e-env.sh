#!/usr/bin/env bash

set -euo pipefail

# Writes a `.env` suitable for local/CI E2E runs where:
# - frontend runs on http://app.localtest.me:5173
# - API runs on http://127.0.0.1:8080 (proxied behind the frontend at /api/*)
#
# Note: Cookie auth requires a non-empty cookie domain; we use `localtest.me`
# so cookies work for `app.localtest.me` without needing /etc/hosts.

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

randstring() {
  # 16 chars is fine for our cookie secret pieces; keep it portable across CI runners.
  openssl rand -base64 48 | tr -d "\n" | tr -d "=+/" | cut -c1-"${1:-16}"
}

cat > .env <<EOF
DATABASE_URL='postgresql://hatchet:hatchet@127.0.0.1:5431/hatchet'

SERVER_ENCRYPTION_MASTER_KEYSET_FILE=./hack/dev/encryption-keys/master.key
SERVER_ENCRYPTION_JWT_PRIVATE_KEYSET_FILE=./hack/dev/encryption-keys/private_ec256.key
SERVER_ENCRYPTION_JWT_PUBLIC_KEYSET_FILE=./hack/dev/encryption-keys/public_ec256.key

SERVER_PORT=8080
# This is used for redirects (e.g. auth middleware). In E2E, the browser origin is the frontend.
SERVER_URL=http://app.localtest.me:5173

# Cookie auth requires a domain; also we run over http so cookies must not be Secure.
SERVER_AUTH_COOKIE_SECRETS="$(randstring 16) $(randstring 16)"
SERVER_AUTH_COOKIE_DOMAIN=localtest.me
SERVER_AUTH_COOKIE_INSECURE=true
SERVER_AUTH_SET_EMAIL_VERIFIED=true

SERVER_MSGQUEUE_KIND=rabbitmq
SERVER_MSGQUEUE_RABBITMQ_URL=amqp://user:password@127.0.0.1:5672/

SERVER_ADDITIONAL_LOGGERS_QUEUE_LEVEL=warn
SERVER_ADDITIONAL_LOGGERS_QUEUE_FORMAT=console
SERVER_ADDITIONAL_LOGGERS_PGXSTATS_LEVEL=error
SERVER_ADDITIONAL_LOGGERS_PGXSTATS_FORMAT=console
SERVER_LOGGER_LEVEL=error
SERVER_LOGGER_FORMAT=console
DATABASE_LOGGER_LEVEL=error
DATABASE_LOGGER_FORMAT=console

SERVER_GRPC_BROADCAST_ADDRESS=127.0.0.1:7070
SERVER_GRPC_INSECURE=true
SERVER_INTERNAL_CLIENT_BASE_STRATEGY=none
SERVER_INTERNAL_CLIENT_BASE_INHERIT_BASE=false
EOF

echo "Wrote $ROOT_DIR/.env"
