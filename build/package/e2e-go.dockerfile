# Base Go environment
# -------------------
FROM golang:1.25-alpine as base
WORKDIR /hatchet

COPY go.mod go.sum ./

RUN go mod download

COPY /pkg ./pkg
COPY /internal ./internal
COPY /api ./api
COPY /sdks/go ./sdks/go

# Go build environment
# --------------------
FROM base AS build-go

RUN go test -c -tags e2e -v -o ./bin/e2e-test ./sdks/go/e2e/

# Deployment environment
# ----------------------
FROM alpine AS deployment

WORKDIR /hatchet

RUN apk update && apk add --no-cache ca-certificates tzdata

COPY --from=build-go /hatchet/bin/e2e-test /hatchet/
