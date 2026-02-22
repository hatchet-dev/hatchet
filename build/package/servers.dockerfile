# Base Go environment
# -------------------
FROM golang:1.25-alpine as base
WORKDIR /hatchet

RUN apk update && apk add --no-cache gcc musl-dev git protoc protobuf-dev

RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2
RUN go install github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen@v2.0.0

COPY go.mod go.sum ./

RUN go mod download

COPY /api ./api
COPY /api-contracts ./api-contracts
COPY /internal ./internal
COPY /pkg ./pkg
COPY /hack ./hack
COPY /cmd ./cmd

RUN go generate ./...

# OpenAPI bundle environment (uses openapi-core only to avoid Redoc/React/styled-components)
# ----------------------------------------------------------------------------------------
FROM node:22-alpine AS build-openapi
WORKDIR /openapi

COPY /api-contracts/openapi ./openapi
COPY /hack/oas/bundle-openapi.mjs ./bundle-openapi.mjs

RUN echo '{ "type": "module", "dependencies": { "@redocly/openapi-core": "2.14.7", "yaml": "2.7.0" } }' > package.json && \
    npm install && \
    node bundle-openapi.mjs && \
    rm -f package.json package-lock.json bundle-openapi.mjs

# Go build environment
# --------------------
FROM base AS build-go

ARG VERSION=v0.1.0-alpha.0

# can be set to "api", "engine", "admin" or "lite"
ARG SERVER_TARGET

# check if the target is empty or not set to api, engine, lite, or admin
RUN if [ -z "$SERVER_TARGET" ] || [ "$SERVER_TARGET" != "api" ] && [ "$SERVER_TARGET" != "engine" ] && [ "$SERVER_TARGET" != "admin" ] && [ "$SERVER_TARGET" != "lite" ] && [ "$SERVER_TARGET" != "migrate" ]; then \
    echo "SERVER_TARGET must be set to 'api', 'engine', 'admin', 'lite', or 'migrate'"; \
    exit 1; \
    fi

RUN sh ./hack/proto/proto.sh

COPY --from=build-openapi /openapi/bin/oas/openapi.yaml ./bin/oas/openapi.yaml

# build oapi
RUN oapi-codegen -config ./api/v1/server/oas/gen/codegen.yaml ./bin/oas/openapi.yaml

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=$GOPATH/pkg/mod \
    go build -ldflags="-w -s -X 'main.Version=${VERSION}'" -a -o ./bin/hatchet-${SERVER_TARGET} ./cmd/hatchet-${SERVER_TARGET}

# Deployment environment
# ----------------------
FROM alpine AS deployment

# can be set to "api", "engine", "admin" or "lite"
ARG SERVER_TARGET=engine
ENV SERVER_TARGET=${SERVER_TARGET}

WORKDIR /hatchet

# openssl and bash needed for admin build
RUN apk update && apk add --no-cache gcc musl-dev openssl bash ca-certificates tzdata

COPY --from=build-go /hatchet/bin/hatchet-${SERVER_TARGET} /hatchet/

# NOTE: this is just here for backwards compatibility with old migrate images which require the atlas-apply.sh script.
# This script is just a wrapped for `/hatchet/hatchet-migrate`.
COPY /hack/db/atlas-apply.sh ./atlas-apply.sh
RUN chmod +x ./atlas-apply.sh

EXPOSE 8080

CMD ["/bin/sh", "-c", "/hatchet/hatchet-${SERVER_TARGET}"]
