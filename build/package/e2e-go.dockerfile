# Base Go environment
# -------------------
FROM golang:1.26-alpine as base
WORKDIR /hatchet

COPY go.mod go.sum ./

# Module downloads fail now and then on a dropped proxy stream; a retry inside
# the image build is the only place that can catch it.
RUN n=0; until go mod download; do n=$((n + 1)); [ "$n" -ge 3 ] && exit 1; sleep $((n * 10)); done

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

CMD ["/hatchet/e2e-test", "-test.v", "-test.timeout=10m"]
