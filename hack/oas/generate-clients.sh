#!/bin/bash

set -eux

ROOT_DIR=$(pwd)

go run github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen@v2.0.0 -config ./pkg/client/rest/codegen.yaml ./bin/oas/openapi.yaml

cd frontend && (npx swagger-typescript-api -p ../bin/oas/openapi.yaml -o ./app/src/lib/api/generated -n hatchet.ts --modular --axios)

cd $ROOT_DIR

cd typescript-sdk && (npx swagger-typescript-api -p ../bin/oas/openapi.yaml -o ./src/clients/rest/generated -n hatchet.ts --modular --axios)

cd $ROOT_DIR

dst_dir=./python-sdk/hatchet_sdk/clients/rest

mkdir -p $dst_dir

tmp_dir=./python-sdk/tmp

# generate into tmp folder
openapi-generator-cli generate -i ./bin/oas/openapi.yaml -g python -o ./python-sdk/tmp --skip-validate-spec \
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


openapi-generator-cli generate -i ./bin/oas/openapi.yaml -g python -o ./python-sdk --skip-validate-spec \
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