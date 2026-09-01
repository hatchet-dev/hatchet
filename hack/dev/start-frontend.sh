#!/bin/bash

set -eux

ARGS=()

if [ -n "${VITE_DEV_HOST:-}" ]; then
	ARGS=(-- --host "$VITE_DEV_HOST")
fi

cd ./frontend/app && npm run dev "${ARGS[@]}"
