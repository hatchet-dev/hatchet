# This expects the hatchet-api image to be built and available on the machine
# -------------------
ARG HATCHET_API_IMAGE

# Stage 1: copy from the existing Go built image
FROM $HATCHET_API_IMAGE AS api-binary-base

# Stage 2: build the frontend
FROM node:22-alpine AS frontend-build

WORKDIR /app

COPY ./frontend/app/package.json ./frontend/app/pnpm-lock.yaml ./
RUN corepack pnpm@10.16.1 --version
RUN corepack pnpm@10.16.1 install --frozen-lockfile && corepack pnpm@10.16.1 store prune

COPY ./frontend/app ./

RUN npm run build

# Stage 3: build the static fileserver
FROM golang:1.26-alpine AS staticfileserver

WORKDIR /app

ENV CGO_ENABLED=0

ARG FIPS=false

COPY go.mod go.sum ./
COPY ./cmd/hatchet-staticfileserver/ ./cmd/hatchet-staticfileserver/

RUN if [ "$FIPS" = "true" ]; then export GOFIPS140=v1.0.0; LDFLAGS="-w"; else LDFLAGS="-w -s"; fi && \
    go build -ldflags="${LDFLAGS}" -a -o hatchet-staticfileserver ./cmd/hatchet-staticfileserver && \
    if [ "$FIPS" = "true" ]; then \
      go version -m ./hatchet-staticfileserver | grep -Ec 'GOFIPS140=v1.0.0|DefaultGODEBUG=.*fips140=only' | grep -qx 2 || { echo "hatchet-staticfileserver is not linked against the validated FIPS module"; exit 1; }; \
    fi

# Stage 4: deployment image
FROM alpine AS deployment

ARG FIPS=false
LABEL run.hatchet.fips=${FIPS}

ENV BASE_PATH=/

WORKDIR /hatchet

RUN apk update && apk add --no-cache ca-certificates tzdata

COPY --from=api-binary-base /hatchet/hatchet-api ./hatchet-api
COPY --from=staticfileserver /app/hatchet-staticfileserver ./hatchet-staticfileserver
COPY --from=frontend-build /app/dist ./html
COPY ./build/package/dashboard-entrypoint.sh ./entrypoint.sh

RUN chmod +x ./entrypoint.sh

EXPOSE 80

CMD ["./entrypoint.sh"]
