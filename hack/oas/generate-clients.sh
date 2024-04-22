#!/bin/bash

set -eux

go run github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen@v2.0.0 -config ./pkg/client/rest/codegen.yaml ./bin/oas/openapi.yaml

cd frontend && (npx swagger-typescript-api -p ../bin/oas/openapi.yaml -o ./app/src/lib/api/generated -n hatchet.ts --modular --axios)
