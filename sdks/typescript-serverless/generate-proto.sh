#!/bin/bash
#
# Generates the TypeScript bindings for the serverless wire contract, v1/serverless.proto, and
# the messages it imports. Run from this directory (pnpm run generate-proto).
#
# useExactTypes=false matches sdks/typescript/generate-protoc.sh so that fromJSON and toJSON
# produce the protojson encoding the operator uses. Service stubs are left out: the package
# never speaks gRPC, and the stubs would pull in nice-grpc types the package does not depend
# on. api-contracts/dispatcher is a second proto path because v1/serverless.proto imports the
# package-less dispatcher.proto under that name, the same way the Go build compiles it
# (hack/proto/proto.sh).

set -euo pipefail

cd "$(dirname "$0")"

OUT_DIR="./src/generated/proto"
IN_DIR="../../api-contracts"
VENDOR_DIR="../../hack/proto/vendor"

rm -rf "$OUT_DIR"
mkdir -p "$OUT_DIR"

./node_modules/.bin/grpc_tools_node_protoc \
  --plugin=protoc-gen-ts_proto=./node_modules/.bin/protoc-gen-ts_proto \
  --ts_proto_out="$OUT_DIR" \
  --ts_proto_opt=outputServices=false,useExactTypes=false \
  --proto_path="$IN_DIR" \
  --proto_path="$IN_DIR/dispatcher" \
  --proto_path="$VENDOR_DIR" \
  v1/serverless.proto \
  v1/dispatcher.proto \
  v1/workflows.proto \
  v1/shared/condition.proto \
  v1/shared/trigger.proto \
  dispatcher.proto
