#!/bin/bash
# This scripts generates local encryption keys for development.

set -eux

ENCRYPTION_KEYS_DIR=./encryption-keys

# Read CERTS_DIR from args if exists
if [ -n "$1" ]; then
    ENCRYPTION_KEYS_DIR=$1
fi

mkdir -p $ENCRYPTION_KEYS_DIR

# Generate a master encryption key
go run ./cmd/hatchet-admin keyset create-local-keys --key-dir $ENCRYPTION_KEYS_DIR
