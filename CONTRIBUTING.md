## Development Setup

> **Note:** this guide assumes you're using MacOS. We simply don't have the bandwidth to test local development on native Windows. Most distros of Linux should work and we would like to support them, so please file an issue if running into an issue with a common distro.

### Prerequisites

- `go 1.21+`
- `docker-compose`
- [`Taskfile`](https://taskfile.dev/installation/)
- The following additional devtools:
  - `protoc`: `brew install protobuf`
  - `caddy` and `nss`: `brew install caddy nss`

### Setup

1. Spin up Postgres and RabbitMQ: `docker-compose up -d`

2. Run `npm install` inside of `./frontend/app`.

3. Generate certificates needed for communicating between the Hatchet client and engine: `task generate-certs`

4. Create environment variables:

```sh
alias randstring='f() { openssl rand -base64 69 | tr -d "\n" | tr -d "=+/" | cut -c1-$1 };f'

cat > .env <<EOF
DATABASE_URL='postgresql://hatchet:hatchet@127.0.0.1:5431/hatchet'
SERVER_TLS_CERT_FILE=./hack/dev/certs/cluster.pem
SERVER_TLS_KEY_FILE=./hack/dev/certs/cluster.key
SERVER_TLS_ROOT_CA_FILE=./hack/dev/certs/ca.cert

SERVER_PORT=8080
SERVER_URL=https://app.dev.hatchet-tools.com

SERVER_AUTH_COOKIE_SECRETS="$(randstring 16) $(randstring 16)"
SERVER_AUTH_COOKIE_DOMAIN=app.dev.hatchet-tools.com
SERVER_AUTH_COOKIE_INSECURE=false
EOF
```

5. Migrate the database: `task prisma-migrate`

6. Generate all files: `task generate`

7. Seed the database: `task seed-dev`

8. Start the Hatchet engine, API server, dashboard, and Prisma studio:

```sh
task start-dev
```

10. To create and test workflows, run the examples in the `./examples` directory. You will need to add the tenant (output from the `task seed-dev` command) to the `.env` file in each example directory.
