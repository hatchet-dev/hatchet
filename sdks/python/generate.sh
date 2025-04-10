#!/bin/bash
#
# Builds python auto-generated protobuf files

set -eux

# deps
version=7.3.0

openapi-generator-cli version || npm install @openapitools/openapi-generator-cli -g

# if [ "$(openapi-generator-cli version)" != "$version" ]; then
#   version-manager set "$version"
# fi

# generate deps from hatchet repo
(cd ../.. && sh hack/oas/generate-server.sh)

# generate python rest client

dst_dir=./hatchet_sdk/clients/rest

mkdir -p $dst_dir

tmp_dir=./tmp

# generate into tmp folder
openapi-generator-cli generate -i ../../bin/oas/openapi.yaml -g python -o ./tmp --skip-validate-spec \
    --global-property=apiTests=false \
    --global-property=apiDocs=true \
    --global-property=modelTests=false \
    --global-property=modelDocs=true \
    --package-name hatchet_sdk.clients.rest

mv $tmp_dir/hatchet_sdk/clients/rest/api_client.py $dst_dir/api_client.py
mv $tmp_dir/hatchet_sdk/clients/rest/configuration.py $dst_dir/configuration.py
mv $tmp_dir/hatchet_sdk/clients/rest/api_response.py $dst_dir/api_response.py
mv $tmp_dir/hatchet_sdk/clients/rest/exceptions.py $dst_dir/exceptions.py
mv $tmp_dir/hatchet_sdk/clients/rest/__init__.py $dst_dir/__init__.py
mv $tmp_dir/hatchet_sdk/clients/rest/rest.py $dst_dir/rest.py

openapi-generator-cli generate -i ../../bin/oas/openapi.yaml -g python -o . --skip-validate-spec \
    --global-property=apis,models \
    --global-property=apiTests=false \
    --global-property=apiDocs=false \
    --global-property=modelTests=false \
    --global-property=modelDocs=false \
    --package-name hatchet_sdk.clients.rest

# copy the __init__ files from tmp to the destination since they are not generated for some reason
cp $tmp_dir/hatchet_sdk/clients/rest/models/__init__.py $dst_dir/models/__init__.py
cp $tmp_dir/hatchet_sdk/clients/rest/api/__init__.py $dst_dir/api/__init__.py

# remove tmp folder
rm -rf $tmp_dir


MIN_GRPCIO_VERSION=$(grep -A 1 'grpcio =' pyproject.toml | grep 'version' | sed -E 's/.*">=([0-9]+\.[0-9]+\.[0-9]+).*/\1/' | sort -V | head -n 1
)

poetry add "grpcio@$MIN_GRPCIO_VERSION" "grpcio-tools@$MIN_GRPCIO_VERSION"

proto_paths=(
  "../../api-contracts/dispatcher dispatcher.proto"
  "../../api-contracts/events events.proto"
  "../../api-contracts/workflows workflows.proto"
  "../../api-contracts v1/shared/condition.proto"
  "../../api-contracts v1/dispatcher.proto"
  "../../api-contracts v1/workflows.proto"
)

for entry in "${proto_paths[@]}"; do
  proto_path=$(echo "$entry" | cut -d' ' -f1)
  proto_file=$(echo "$entry" | cut -d' ' -f2-)

  echo "Generating Python code for $proto_file with proto_path=$proto_path"

  poetry run python -m grpc_tools.protoc \
    --proto_path="$proto_path" \
    --python_out=./hatchet_sdk/contracts \
    --pyi_out=./hatchet_sdk/contracts \
    --grpc_python_out=./hatchet_sdk/contracts \
    "$proto_file"
done

## Hack to fix broken import paths with absolute paths
if [[ "$OSTYPE" == "darwin"* ]]; then
  find ./hatchet_sdk/contracts -type f -name '*.py.*' -exec sed -i '' 's/from v1/from hatchet_sdk.contracts.v1/g' {} +
else
  find ./hatchet_sdk/contracts -type f -name '*.py.*' -exec sed -i 's/from v1/from hatchet_sdk.contracts.v1/g' {} +
fi

git restore pyproject.toml poetry.lock

poetry install --all-extras

# Fix relative imports in _grpc.py files
find ./hatchet_sdk/contracts -type f -name '*_grpc.py' -print0 | xargs -0 sed -i '' 's/from v1/from hatchet_sdk.contracts.v1/g'
find ./hatchet_sdk/contracts -type f -name '*_pb2.pyi' -print0 | xargs -0 sed -i '' 's/from v1/from hatchet_sdk.contracts.v1/g'
find ./hatchet_sdk/contracts -type f -name '*_pb2.py' -print0 | xargs -0 sed -i '' 's/from v1/from hatchet_sdk.contracts.v1/g'

find ./hatchet_sdk/contracts -type f -name '*_grpc.py' -print0 | xargs -0 sed -i '' 's/import dispatcher_pb2 as dispatcher__pb2/from hatchet_sdk.contracts import dispatcher_pb2 as dispatcher__pb2/g'
find ./hatchet_sdk/contracts -type f -name '*_grpc.py' -print0 | xargs -0 sed -i '' 's/import events_pb2 as events__pb2/from hatchet_sdk.contracts import events_pb2 as events__pb2/g'
find ./hatchet_sdk/contracts -type f -name '*_grpc.py' -print0 | xargs -0 sed -i '' 's/import workflows_pb2 as workflows__pb2/from hatchet_sdk.contracts import workflows_pb2 as workflows__pb2/g'

find ./hatchet_sdk/contracts -type f -name '*_grpc.py' -print0 | xargs -0 sed -i '' 's/def __init__(self, channel):/def __init__(self, channel: grpc.Channel | grpc.aio.Channel) -> None:/g'

set +e
# Note: Hack to run the linters without failing so that we can apply the patch
./lint.sh
set -e

# apply patch to openapi-generator generated code
patch -p1 --no-backup-if-mismatch <./openapi_patch.patch

# Rerun the linters and fail if there are any issues
./lint.sh
