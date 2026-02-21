#!/usr/bin/env bash

set -euo pipefail

# Directory to write generated code to (.js and .d.ts files)

OUT_DIR="./src/protoc"

if [ -d "./hatchet" ]; then
    IN_DIR="./hatchet/api-contracts"
else
    IN_DIR="../../api-contracts"
fi

# Generate code.
# Prefer system `protoc` (available in Nix environments) over `grpc-tools`' bundled protoc.
PROTOC_BIN="${PROTOC_BIN:-}"
if [ -z "$PROTOC_BIN" ]; then
  if command -v protoc >/dev/null 2>&1; then
    PROTOC_BIN="protoc"
  else
    PROTOC_BIN="./node_modules/.bin/grpc_tools_node_protoc"
  fi
fi

"$PROTOC_BIN" \
  --plugin=protoc-gen-ts_proto=./node_modules/.bin/protoc-gen-ts_proto \
  --ts_proto_out="$OUT_DIR" \
  --ts_proto_opt=outputServices=nice-grpc,outputServices=generic-definitions,useExactTypes=false \
  --proto_path="$IN_DIR" \
  $(find "$IN_DIR" -type f -name '*.proto' -print)

pnpm lint:fix
