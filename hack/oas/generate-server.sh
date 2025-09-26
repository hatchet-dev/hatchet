#!/bin/bash

set -eux

cat << EOF > package.json
{
    "name": "hack-for-redocly-cli-bug",
    "version": "1.0.0",
    "dependencies": {
        "@redocly/cli": "latest"
    },
    "overrides": {
        "mobx-react": "9.2.0"
    }
}
EOF

npx --yes @redocly/cli bundle ./api-contracts/openapi/openapi.yaml --output ./bin/oas/openapi.yaml --ext yaml
go run github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen@v2.0.0 -config ./api/v1/server/oas/gen/codegen.yaml ./bin/oas/openapi.yaml

rm package.json