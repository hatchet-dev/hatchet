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

ARG FIPS=false

RUN if [ "$FIPS" = "true" ]; then export GOFIPS140=v1.0.0; LDFLAGS="-w"; else LDFLAGS="-w -s"; fi && \
    go build -ldflags="${LDFLAGS}" -a -o ./bin/hatchet-load-test ./cli && \
    go build -ldflags="${LDFLAGS}" -a -o ./bin/hatchet-load-test-worker ./cli/go && \
    if [ "$FIPS" = "true" ]; then \
      for b in hatchet-load-test hatchet-load-test-worker; do \
        go version -m ./bin/$b | grep -Ec 'GOFIPS140=v1.0.0|DefaultGODEBUG=.*fips140=only' | grep -qx 2 || { echo "$b is not linked against the validated FIPS module"; exit 1; }; \
      done; \
    fi

# Deployment environment
# ----------------------
FROM alpine AS deployment

ARG FIPS=false
LABEL run.hatchet.fips=${FIPS}

WORKDIR /hatchet

# openssl and bash needed for admin build
RUN apk update && apk add --no-cache openssl bash ca-certificates tzdata

COPY --from=build-go /hatchet/bin/hatchet-load-test /hatchet/
COPY --from=build-go /hatchet/bin/hatchet-load-test-worker /hatchet/

CMD /hatchet/hatchet-load-test
