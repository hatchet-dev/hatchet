# Base Go environment
# -------------------
FROM golang:1.26-alpine as base
WORKDIR /hatchet

ENV CGO_ENABLED=0

COPY go.mod go.sum ./

RUN go mod download

COPY /pkg ./pkg
COPY /internal ./internal
COPY /api ./api
COPY /sdks/go ./sdks/go
COPY /cmd/hatchet-loadtest ./cli

# Go build environment
# --------------------
FROM base AS build-go

RUN go build -ldflags="-w -s" -a -o ./bin/hatchet-load-test ./cli
RUN go build -ldflags="-w -s" -a -o ./bin/hatchet-load-test-worker ./cli/go

# Deployment environment
# ----------------------
FROM alpine AS deployment

WORKDIR /hatchet

# openssl and bash needed for admin build
RUN apk update && apk add --no-cache openssl bash ca-certificates tzdata

COPY --from=build-go /hatchet/bin/hatchet-load-test /hatchet/
COPY --from=build-go /hatchet/bin/hatchet-load-test-worker /hatchet/

CMD /hatchet/hatchet-load-test
