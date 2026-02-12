#!/bin/bash
#
# Generates Ruby protobuf and gRPC service stubs from Hatchet API contracts.
# Requires: gem install grpc-tools
#
# Usage: cd sdks/ruby && bash generate.sh

set -eux

CONTRACTS_DIR="../../api-contracts"
OUTPUT_DIR="./src/lib/hatchet/contracts"

# Ensure output directories exist
mkdir -p "$OUTPUT_DIR/dispatcher"
mkdir -p "$OUTPUT_DIR/events"
mkdir -p "$OUTPUT_DIR/workflows"
mkdir -p "$OUTPUT_DIR/v1/shared"

# Proto files to generate (proto_path proto_file)
proto_entries=(
  "dispatcher/dispatcher.proto"
  "events/events.proto"
  "workflows/workflows.proto"
  "v1/shared/condition.proto"
  "v1/dispatcher.proto"
  "v1/workflows.proto"
)

for proto_file in "${proto_entries[@]}"; do
  echo "Generating Ruby code for $proto_file"

  grpc_tools_ruby_protoc \
    --proto_path="$CONTRACTS_DIR" \
    --ruby_out="$OUTPUT_DIR" \
    --grpc_out="$OUTPUT_DIR" \
    "$proto_file"
done

echo "Ruby protobuf generation complete. Output in $OUTPUT_DIR"
