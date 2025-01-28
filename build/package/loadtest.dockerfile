# Base Go environment
# -------------------
FROM golang:1.23-alpine as base
WORKDIR /hatchet

COPY go.mod go.sum ./

RUN go mod download

COPY /pkg ./pkg
COPY /internal ./internal
COPY /api ./api
COPY /examples/loadtest/cli ./cli

# Go build environment
# --------------------
FROM base AS build-go

RUN go build -a -o ./bin/hatchet-load-test ./cli

# Deployment environment
# ----------------------
FROM alpine AS deployment

WORKDIR /hatchet

# openssl and bash needed for admin build
RUN apk update && apk add --no-cache gcc musl-dev openssl bash ca-certificates

COPY --from=build-go /hatchet/bin/hatchet-load-test /hatchet/

CMD /hatchet/hatchet-load-test
    