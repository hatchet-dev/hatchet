# Contributing

### Setup

1. Make sure all prerequisite dependencies are installed:

   - [Go 1.24+](https://go.dev/doc/install)
   - [Node.js v18+](https://nodejs.org/en/download) - we recommend using [nvm](https://github.com/nvm-sh/nvm) for managing node versions.
   - [pnpm](https://pnpm.io/installation) installed globally (`npm i -g pnpm`)
   - [Docker Desktop](https://docs.docker.com/desktop/install/mac-install/)
   - [protoc](https://grpc.io/docs/protoc-installation/)
   - [pip](https://pip.pypa.io/en/stable/installation/)
   - [Caddy](https://caddyserver.com/docs/install)
   - [atlas](https://atlasgo.io/)
   - [pre-commit](https://pre-commit.com/)
     - You can install this in a virtual environment with `python3 -m venv venv && source venv/bin/activate && pip3 install pre-commit`

1. You can then populate a local `.env` file with the following:

```
DATABASE_URL='postgresql://hatchet:hatchet@127.0.0.1:5431/hatchet'

SERVER_ENCRYPTION_MASTER_KEYSET_FILE=./hack/dev/encryption-keys/master.key
SERVER_ENCRYPTION_JWT_PRIVATE_KEYSET_FILE=./hack/dev/encryption-keys/private_ec256.key
SERVER_ENCRYPTION_JWT_PUBLIC_KEYSET_FILE=./hack/dev/encryption-keys/public_ec256.key

SERVER_PORT=8080
SERVER_URL=http://localhost:8080

SERVER_AUTH_COOKIE_SECRETS="1234"
SERVER_AUTH_COOKIE_DOMAIN=app.dev.hatchet-tools.com
SERVER_AUTH_COOKIE_INSECURE=false
SERVER_AUTH_SET_EMAIL_VERIFIED=true

SERVER_MSGQUEUE_KIND=rabbitmq
SERVER_MSGQUEUE_RABBITMQ_URL=amqp://user:password@127.0.0.1:5672/

SERVER_GRPC_BROADCAST_ADDRESS=grpc.dev.hatchet-tools.com:443
SERVER_GRPC_INSECURE=true
```

1. Start the Database and RabbitMQ services:

```sh
task start-db
```

1. Install dependencies, run migrations, generate encryption keys, and seed the database:

```sh
task setup
```

**_Note: You might need to run this as `sudo` so it can install certificates._**

### Starting the dev server

Start the Hatchet engine, API server, dashboard, and Prisma studio:

might need to run as sudo so it can install deps

```sh
task start-dev # or task start-dev-tmux if you want to use tmux panes
```

### Creating and testing workflows

To create and test workflows, run the examples in the `./examples` directory.

You will need to add the tenant (output from the `task seed-dev` command) to the `.env` file in each example directory. An example `.env` file for the `./examples/simple` directory. You can be generated and add it to the .env file via:

```sh
cat >> ./examples/simple/.env <<EOF
HATCHET_CLIENT_TENANT_ID=707d0855-80ab-4e1f-a156-f1c4546cbf52
HATCHET_CLIENT_TLS_ROOT_CA_FILE=../../hack/dev/certs/ca.cert
HATCHET_CLIENT_TLS_SERVER_NAME=cluster
HATCHET_CLIENT_TOKEN="$(go run ./cmd/hatchet-admin token create --name local --tenant-id 707d0855-80ab-4e1f-a156-f1c4546cbf52)"
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

## Working with Database Models

1. Create or modify the required SQL queries in `./pkg/repository/prisma/dbsqlc`
2. Add new queries files to `./pkg/repository/prisma/dbsqlc/sqlc.yaml`
3. Make schema changes in `./sql/schema/schema.sql`
4. Create a new migration file with `task migrate`
5. Rename the migration file in `./sql/migrations/` to match the latest tag.
6. Generate Go with `task generate-all`
7. Run `atlas migrate hash --dir "file://sql/migrations"` to generate the atlas hash.
