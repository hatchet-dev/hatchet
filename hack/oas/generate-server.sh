#!/bin/bash

set -eux

npx --yes swagger-cli bundle ./api-contracts/openapi/openapi.yaml --outfile bin/oas/openapi.yaml --type yaml
go run github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen@latest -config ./api/v1/server/oas/gen/codegen.yaml ./bin/oas/openapi.yaml
