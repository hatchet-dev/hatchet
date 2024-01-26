## Development Setup

> **Note:** this guide assumes you're using MacOS. We simply don't have the bandwidth to test local development on native Windows. Most distros of Linux should work and we would like to support them, so please file an issue if running into an issue with a common distro.

### Prerequisites

- `go 1.21+`
- `docker-compose`
- [`Taskfile`](https://taskfile.dev/installation/)
- The following additional devtools:
  - `protoc`: `brew install protobuf@25`
  - `caddy` and `nss`: `brew install caddy nss`

### Setup

1. Spin up Postgres and RabbitMQ: `docker-compose up -d`

2. Run `pnpm install` inside of `./frontend/app`.

3. Generate certificates needed for communicating between the Hatchet client and engine: `task generate-certs`

4. Generate keysets for encryption: `task generate-local-encryption-keys`

5. Create environment variables:

```sh
alias randstring='f() { openssl rand -base64 69 | tr -d "\n" | tr -d "=+/" | cut -c1-$1 };f'

cat > .env <<EOF
DATABASE_URL='postgresql://hatchet:hatchet@127.0.0.1:5431/hatchet'
SERVER_TLS_CERT_FILE=./hack/dev/certs/cluster.pem
SERVER_TLS_KEY_FILE=./hack/dev/certs/cluster.key
SERVER_TLS_ROOT_CA_FILE=./hack/dev/certs/ca.cert

SERVER_ENCRYPTION_MASTER_KEYSET_FILE=./hack/dev/encryption-keys/master.key
SERVER_ENCRYPTION_JWT_PRIVATE_KEYSET_FILE=./hack/dev/encryption-keys/private_ec256.key
SERVER_ENCRYPTION_JWT_PUBLIC_KEYSET_FILE=./hack/dev/encryption-keys/public_ec256.key

SERVER_PORT=8080
SERVER_URL=https://app.dev.hatchet-tools.com

SERVER_AUTH_COOKIE_SECRETS="$(randstring 16) $(randstring 16)"
SERVER_AUTH_COOKIE_DOMAIN=app.dev.hatchet-tools.com
SERVER_AUTH_COOKIE_INSECURE=false
SERVER_AUTH_SET_EMAIL_VERIFIED=true

SERVER_LOGGER_LEVEL=debug
SERVER_LOGGER_FORMAT=console
DATABASE_LOGGER_LEVEL=debug
DATABASE_LOGGER_FORMAT=console
EOF
```

6. Migrate the database: `task prisma-migrate`

7. Generate all files: `task generate`

8. Seed the database: `task seed-dev`

9. Start the Hatchet engine, API server, dashboard, and Prisma studio:

```sh
task start-dev
```

### Creating and testing workflows

To create and test workflows, run the examples in the `./examples` directory.

You will need to add the tenant (output from the `task seed-dev` command) to the `.env` file in each example directory. An example `.env` file for the `./examples/simple` directory can be generated via:

```sh
alias get_token='go run ./cmd/hatchet-admin token create --name local --tenant-id 707d0855-80ab-4e1f-a156-f1c4546cbf52'

cat > ./examples/simple/.env <<EOF
HATCHET_CLIENT_TENANT_ID=707d0855-80ab-4e1f-a156-f1c4546cbf52
HATCHET_CLIENT_TLS_ROOT_CA_FILE=../../hack/dev/certs/ca.cert
HATCHET_CLIENT_TLS_SERVER_NAME=cluster
HATCHET_CLIENT_TOKEN="$(get_token)"
EOF
```

This example can then be run via `go run main.go` from the `./examples/simple` directory.

### Logging

You can set the following logging formats to configure your logging:

```
# info, debug, error, etc
SERVER_LOGGER_LEVEL=debug

# json or console
SERVER_LOGGER_FORMAT=json

DATABASE_LOGGER_LEVEL=debug
DATABASE_LOGGER_FORMAT=console
```

### OpenTelemetry

You can set the following to enable distributed tracing:

```
SERVER_OTEL_SERVICE_NAME=engine
SERVER_OTEL_COLLECTOR_URL=<collector-url>

# optional
OTEL_EXPORTER_OTLP_HEADERS=<optional-headers>

# optional
OTEL_EXPORTER_OTLP_ENDPOINT=<collector-url>
```

### CloudKMS

CloudKMS can be used to generate master encryption keys:

```
gcloud kms keyrings create "development" --location "global"
gcloud kms keys create "development" --location "global" --keyring "development" --purpose "encryption"
gcloud kms keys list --location "global" --keyring "development"
```

From the last step, copy the Key URI and set the following environment variable:

```
SERVER_ENCRYPTION_CLOUDKMS_KEY_URI=gcp-kms://projects/<PROJECT>/locations/global/keyRings/development/cryptoKeys/development
```

Generate a service account in GCP which can encrypt/decrypt on CloudKMS, then download a service account JSON file and set it via:

```
SERVER_ENCRYPTION_CLOUDKMS_CREDENTIALS_JSON='{...}'
```

## Issues

### Query engine leakage

Sometimes the spawned query engines from Prisma don't get killed when hot reloading. You can run `task kill-query-engines` on OSX to kill the query engines.

Make sure you call `.Disconnect` on the database config object when writing CLI commands which interact with the database. If you don't, and you try to wrap these CLI commands in a new command, it will never exit, for example:

```
export HATCHET_CLIENT_TOKEN="$(go run ./cmd/hatchet-admin token create --tenant-id <tenant>)"
```
