#!/bin/bash

swagger-cli bundle ./api-contracts/openapi/openapi.yaml --outfile bin/oas/openapi.yaml --type yaml
oapi-codegen -config ./api/v1/server/oas/gen/codegen.yaml ./bin/oas/openapi.yaml
