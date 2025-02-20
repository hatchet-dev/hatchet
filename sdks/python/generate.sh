#!/bin/bash
#
# Builds python auto-generated protobuf files

set -eux

ROOT_DIR=$(pwd)

# deps
version=7.3.0

openapi-generator-cli version || npm install @openapitools/openapi-generator-cli -g

# if [ "$(openapi-generator-cli version)" != "$version" ]; then
#   version-manager set "$version"
# fi

# generate deps from hatchet repo
cd ../oss/ && sh ./hack/oas/generate-server.sh && cd $ROOT_DIR

# generate python rest client

dst_dir=./hatchet_sdk/clients/rest

mkdir -p $dst_dir

tmp_dir=./tmp

# generate into tmp folder
openapi-generator-cli generate -i ../oss/bin/oas/openapi.yaml -g python -o ./tmp --skip-validate-spec \
    --library asyncio \
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

openapi-generator-cli generate -i ../oss/bin/oas/openapi.yaml -g python -o . --skip-validate-spec \
    --library asyncio \
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

poetry run python -m grpc_tools.protoc --proto_path=../oss/api-contracts/dispatcher --python_out=./hatchet_sdk/contracts --pyi_out=./hatchet_sdk/contracts --grpc_python_out=./hatchet_sdk/contracts dispatcher.proto
poetry run python -m grpc_tools.protoc --proto_path=../oss/api-contracts/events --python_out=./hatchet_sdk/contracts --pyi_out=./hatchet_sdk/contracts --grpc_python_out=./hatchet_sdk/contracts events.proto
poetry run python -m grpc_tools.protoc --proto_path=../oss/api-contracts/workflows --python_out=./hatchet_sdk/contracts --pyi_out=./hatchet_sdk/contracts --grpc_python_out=./hatchet_sdk/contracts workflows.proto

git restore pyproject.toml poetry.lock

# Fix relative imports in _grpc.py files
if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS
    find ./hatchet_sdk/contracts -type f -name '*_grpc.py' -print0 | xargs -0 sed -i '' 's/^import \([^ ]*\)_pb2/from . import \1_pb2/'
else
    # Linux and others
    find ./hatchet_sdk/contracts -type f -name '*_grpc.py' -print0 | xargs -0 sed -i 's/^import \([^ ]*\)_pb2/from . import \1_pb2/'
fi

# ensure that pre-commit is applied without errors
./lint.sh

# apply patch to openapi-generator generated code
patch -p1 --no-backup-if-mismatch <./openapi_patch.patch
