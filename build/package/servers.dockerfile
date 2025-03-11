# Base Go environment
# -------------------
FROM golang:1.23-alpine as base
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

# OpenAPI bundle environment
# -------------------------
FROM node:18-alpine as build-openapi
WORKDIR /openapi

RUN npm install -g npm@8.1 @redocly/cli@latest

COPY /api-contracts/openapi ./openapi

RUN npx @redocly/cli bundle ./openapi/openapi.yaml --output ./bin/oas/openapi.yaml --ext yaml

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

WORKDIR /hatchet

# openssl and bash needed for admin build
RUN apk update && apk add --no-cache gcc musl-dev openssl bash ca-certificates

COPY --from=build-go /hatchet/bin/hatchet-${SERVER_TARGET} /hatchet/

EXPOSE 8080
CMD /hatchet/hatchet-${SERVER_TARGET}
