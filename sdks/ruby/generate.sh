#!/bin/bash
#
# Single entry point for all Ruby SDK code generation.
#
# Generates:
#   1. Protobuf/gRPC stubs  (from api-contracts/*.proto)
#   2. REST API client       (from bin/oas/openapi.yaml via openapi-generator)
#
# Usage:
#   cd sdks/ruby && bash generate.sh          # generate everything
#   cd sdks/ruby && bash generate.sh proto     # protobuf only
#   cd sdks/ruby && bash generate.sh rest      # REST client only
#
# Prerequisites:
#   - grpc-tools gem          (gem install grpc-tools)
#   - openapi-generator-cli   (npm install -g @openapitools/openapi-generator-cli)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# ── Protobuf / gRPC generation ──────────────────────────────────────────────

generate_proto() {
  echo "==> Generating protobuf/gRPC stubs..."

  local contracts_dir="$REPO_ROOT/api-contracts"
  local output_dir="$SCRIPT_DIR/src/lib/hatchet/contracts"

  mkdir -p "$output_dir/dispatcher"
  mkdir -p "$output_dir/events"
  mkdir -p "$output_dir/workflows"
  mkdir -p "$output_dir/v1/shared"

  local proto_files=(
    "dispatcher/dispatcher.proto"
    "events/events.proto"
    "workflows/workflows.proto"
    "v1/shared/condition.proto"
    "v1/dispatcher.proto"
    "v1/workflows.proto"
  )

  for proto_file in "${proto_files[@]}"; do
    echo "    $proto_file"
    grpc_tools_ruby_protoc \
      --proto_path="$contracts_dir" \
      --ruby_out="$output_dir" \
      --grpc_out="$output_dir" \
      "$proto_file"
  done

  echo "    Done."
}

# ── REST API client generation ───────────────────────────────────────────────

generate_rest() {
  echo "==> Generating REST API client from OpenAPI spec..."

  local openapi_spec="$REPO_ROOT/bin/oas/openapi.yaml"
  local output_dir="$SCRIPT_DIR/src/lib/hatchet/clients/rest"
  local config_file="$SCRIPT_DIR/src/config/openapi_generator_config.json"

  if [ ! -f "$openapi_spec" ]; then
    echo "ERROR: OpenAPI spec not found at $openapi_spec" >&2
    exit 1
  fi

  # Install openapi-generator-cli if missing
  if ! command -v openapi-generator-cli &>/dev/null; then
    echo "    Installing openapi-generator-cli..."
    npm install -g @openapitools/openapi-generator-cli
  fi

  # Generate
  local additional_props="gemName=hatchet-sdk-rest,moduleName=HatchetSdkRest,gemVersion=0.0.1,gemDescription=HatchetRubySDKRestClient,gemAuthor=HatchetTeam,gemHomepage=https://github.com/hatchet-dev/hatchet,gemLicense=MIT,library=faraday"
  # TODO-RUBY: we can generate docs here :wow:
  local cmd=(
    openapi-generator-cli generate
    -i "$openapi_spec"
    -g ruby
    -o "$output_dir"
    --skip-validate-spec
    --global-property "apiTests=false,modelTests=false,apiDocs=false,modelDocs=false"
    --additional-properties "$additional_props"
  )

  if [ -f "$config_file" ]; then
    cmd+=(-c "$config_file")
  fi

  "${cmd[@]}"

  # ── Post-generation patches ──────────────────────────────────────────────
  echo "    Applying patches..."
  apply_cookie_auth_patch "$output_dir"

  echo "    Done."
}

# Patch the generated client to support cookie-based auth and skip nil values.
apply_cookie_auth_patch() {
  local output_dir="$1"

  # 1. Fix configuration.rb: fill in empty 'in:' for cookie auth
  local config_rb="$output_dir/lib/hatchet-sdk-rest/configuration.rb"
  if [ -f "$config_rb" ]; then
    sed -i.bak "s/in: ,/in: 'cookie',/g" "$config_rb" && rm -f "$config_rb.bak"
  fi

  # 2. Fix api_client.rb: add cookie support + nil guard
  local api_client_rb="$output_dir/lib/hatchet-sdk-rest/api_client.rb"
  if [ -f "$api_client_rb" ]; then
    ruby -e '
      path = ARGV[0]
      content = File.read(path)

      # Match the auth switch block regardless of indentation
      old_pattern = /^(\s*)case auth_setting\[:in\]\n\s*when '\''header'\'' then header_params\[auth_setting\[:key\]\] = auth_setting\[:value\]\n\s*when '\''query'\''  then query_params\[auth_setting\[:key\]\] = auth_setting\[:value\]\n\s*else fail ArgumentError, '\''Authentication token must be in `query` or `header`'\''\n\s*end/

      if content.match(old_pattern)
        indent = content.match(old_pattern)[1]
        new_auth = "#{indent}next if auth_setting[:value].nil? || auth_setting[:value].to_s.empty?\n" \
                   "#{indent}case auth_setting[:in]\n" \
                   "#{indent}when '\''header'\'' then header_params[auth_setting[:key]] = auth_setting[:value]\n" \
                   "#{indent}when '\''query'\''  then query_params[auth_setting[:key]] = auth_setting[:value]\n" \
                   "#{indent}when '\''cookie'\'' then header_params['\''Cookie'\''] = \"\#{auth_setting[:key]}=\#{auth_setting[:value]}\"\n" \
                   "#{indent}else next\n" \
                   "#{indent}end"
        content.sub!(old_pattern, new_auth)
        File.write(path, content)
        puts "      Patched api_client.rb"
      else
        puts "      api_client.rb already patched (skipping)"
      end
    ' "$api_client_rb"
  fi
}

# ── Main ─────────────────────────────────────────────────────────────────────

case "${1:-all}" in
  proto)  generate_proto ;;
  rest)   generate_rest  ;;
  all)
    generate_proto
    generate_rest
    ;;
  -h|--help)
    echo "Usage: $0 [proto|rest|all]"
    echo "  proto   Generate protobuf/gRPC stubs only"
    echo "  rest    Generate REST API client only"
    echo "  all     Generate everything (default)"
    exit 0
    ;;
  *)
    echo "Unknown command: $1. Use --help for usage." >&2
    exit 1
    ;;
esac

echo "==> All generation complete."
