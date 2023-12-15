#!/bin/bash

cat > .env <<EOF
TEMPORAL_CLIENT_TLS_ROOT_CA_FILE=./hack/dev/certs/ca.cert
TEMPORAL_CLIENT_TLS_CERT_FILE=./hack/dev/certs/client-worker.pem
TEMPORAL_CLIENT_TLS_KEY_FILE=./hack/dev/certs/client-worker.key
TEMPORAL_CLIENT_TLS_SERVER_NAME=cluster

TEMPORAL_SQLITE_PATH=./temporal.db
TEMPORAL_LOG_LEVEL=error

TEMPORAL_NAMESPACES=default

TEMPORAL_FRONTEND_TLS_SERVER_NAME=cluster
TEMPORAL_FRONTEND_TLS_CERT_FILE=./hack/dev/certs/cluster.pem
TEMPORAL_FRONTEND_TLS_KEY_FILE=./hack/dev/certs/cluster.key
TEMPORAL_FRONTEND_TLS_ROOT_CA_FILE=./hack/dev/certs/ca.cert

TEMPORAL_WORKER_TLS_SERVER_NAME=cluster
TEMPORAL_WORKER_TLS_CERT_FILE=./hack/dev/certs/cluster.pem
TEMPORAL_WORKER_TLS_KEY_FILE=./hack/dev/certs/cluster.key
TEMPORAL_WORKER_TLS_ROOT_CA_FILE=./hack/dev/certs/ca.cert

TEMPORAL_INTERNODE_TLS_SERVER_NAME=cluster
TEMPORAL_INTERNODE_TLS_CERT_FILE=./hack/dev/certs/cluster.pem
TEMPORAL_INTERNODE_TLS_KEY_FILE=./hack/dev/certs/cluster.key
TEMPORAL_INTERNODE_TLS_ROOT_CA_FILE=./hack/dev/certs/ca.cert

TEMPORAL_UI_TLS_ROOT_CA_FILE=./hack/dev/certs/ca.cert
TEMPORAL_UI_TLS_CERT_FILE=./hack/dev/certs/client-internal-admin.pem
TEMPORAL_UI_TLS_KEY_FILE=./hack/dev/certs/client-internal-admin.key
TEMPORAL_UI_TLS_SERVER_NAME=cluster
EOF
