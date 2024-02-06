#!/bin/bash

set -eux

ROOT_DIR=$(pwd)

cd frontend && (npx swagger-typescript-api -p ../bin/oas/openapi.yaml -o ./app/src/lib/api/generated -n hatchet.ts --modular --axios)

cd $ROOT_DIR

cd typescript-sdk && (npx swagger-typescript-api -p ../bin/oas/openapi.yaml -o ./src/clients/rest/generated -n hatchet.ts --modular --axios)

cd $ROOT_DIR
