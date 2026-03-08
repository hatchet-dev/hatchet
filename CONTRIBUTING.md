# Contributing

This guide will help you understand how to contribute effectively to the Hatchet project.

## Getting Started

New to Hatchet? Start with our [Architecture](https://docs.hatchet.run/home/architecture) docs to familiarize yourself with Hatchet's core system design.

Then, before contributing, check out the following sections:

- [Development Environment Setup](#development-environment-setup)
- [Pull Requests](#pull-requests)
- [Testing](#testing)
- [Running Locally](#running-locally)
  - [Example Workflow](#example-workflow)

## Development Environment Setup

### Core dependencies

Ensure the following are installed:

- [Go 1.25+](https://go.dev/doc/install)
- [Node.js v24+](https://nodejs.org/en/download) — required for the frontend and TypeScript SDK. We recommend [nvm](https://github.com/nvm-sh/nvm) to match the version in [`.nvmrc`](.nvmrc).
- [pnpm](https://pnpm.io/installation) — `corepack enable pnpm` or `npm install -g pnpm`
- [Docker](https://docs.docker.com/engine/install/)
- [task](https://taskfile.dev/docs/installation)
- [protoc](https://grpc.io/docs/protoc-installation/)
- [Caddy](https://caddyserver.com/docs/install)
- [goose](https://pressly.github.io/goose/installation/)
- [pre-commit](https://pre-commit.com/)
  - You can install this in a virtual environment with `task pre-commit-install`

### SDK dependencies

Each SDK requires its own tooling. Install them using your preferred version manager, then run the corresponding setup task.

| SDK            | Prerequisites                                                                                                                                                                     | Setup                   |
| -------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------- |
| **Go**         | [Go 1.25+](https://go.dev/doc/install) (included in core dependencies above)                                                                                                      | `go mod download`       |
| **TypeScript** | [Node.js v24+](https://nodejs.org/en/download) (we recommend [nvm](https://github.com/nvm-sh/nvm) to match [`.nvmrc`](.nvmrc)), [pnpm](https://pnpm.io/installation)              | `task setup-typescript` |
| **Python**     | [Python 3.14.3](https://www.python.org/downloads/release/python-3143/) (e.g. via [pyenv](https://github.com/pyenv/pyenv)), [Poetry](https://python-poetry.org/docs/#installation) | `task setup-python`     |
| **Ruby**       | [Ruby 3.2+](https://www.ruby-lang.org/en/downloads/) (e.g. via [rbenv](https://github.com/rbenv/rbenv)), [Bundler](https://bundler.io/)                                           | `task setup-ruby`       |

To set up all SDKs at once:

```sh
task setup-sdks
```

Each setup task runs a `check-install-*` step first that verifies the required tools are on your `PATH` and prints a message if anything is missing.

### Linting and formatting

Lint and auto-fix all projects:

```sh
task lint
```

Or lint individual SDKs:

```sh
task lint-go
task lint-python
task lint-typescript
task lint-ruby
```

## Pull Requests

Before opening a PR, check if there's a related issue in our [backlog](https://github.com/hatchet-dev/hatchet/issues).

For non-trivial changes (anything beyond typos or patch version bumps), please create an issue first so we can discuss the proposal and ensure it aligns with the project.

Next, ensure all changes are:

- Unit tested with `task test`
- Linted with `task lint`
- Formatted with `task fmt`
- Integration tested with `task test-integration` (when applicable)

If your changes require documentation updates, modify the relevant files in [`frontend/docs/pages/`](frontend/docs/pages/). You can spin up the documentation site locally by running `task docs`. By default, this will be available at [`http://localhost:3000`](http://localhost:3000).

For configuration changes, see [Updating Configuration](docs/development/updating-configuration.md).

## Testing

Hatchet uses Go build tags to categorize tests into different test suites. For example, these build tags mark a test as unit-only:

```go
//go:build !e2e && !load && !rampup && !integration

func TestMyUnitOfCode() { ... }
```

Most contributors should familiarize themselves with **unit testing** and **integration testing**.

**Unit tests** verify individual functions without external dependencies:

```sh
task test
```

**Integration tests** verify components working together with real dependencies (normally spun up via `docker compose`):

```sh
task test-integration
```

Note: **manual testing** is acceptable for cases where automated testing is impractical, but testing steps should be clearly outlined in your PR description.

## Running locally

1. Start the Postgres Database and RabbitMQ services:

```sh
task start-db
```

2. Install Go & Node.js dependencies, run migrations, generate encryption keys, and seed the database:

```sh
task setup
```

3. Start the Hatchet engine, API server, and frontend:

```sh
task start-dev # or task start-dev-tmux if you want to use tmux panes
```

Once started, you should be able to access the Hatchet UI at [https://app.dev.hatchet-tools.com](https://app.dev.hatchet-tools.com).

### Example Workflow

1. Generate client credentials:

```sh
task init-dev-env | tee ./examples/go/simple/.env
```

2. Run the simple workflow by loading the environment variables from `./examples/go/simple/.env`:

```sh
cd ./examples/go/simple
env $(cat .env | xargs) go run main.go
```

You should see the following logs if the workflow was started against your local instance successfully:

```log
{"level":"debug","service":"client","message":"connecting to 127.0.0.1:7070 without TLS"}
{"level":"info","service":"client","message":"gzip compression enabled for gRPC client"}
{"level":"debug","service":"worker","message":"worker simple-worker is listening for actions: [process-message:process-message]"}
{"level":"debug","service":"client","message":"No compute configs found, skipping cloud registration and running all actions locally."}
{"level":"debug","service":"client","message":"Registered worker with id: c47cc839-8c3b-4b0f-a904-00e37f164b7d"}
{"level":"debug","service":"client","message":"Starting to listen for actions"}
{"level":"debug","service":"client","message":"updating worker c47cc839-8c3b-4b0f-a904-00e37f164b7d heartbeat"}
```

## Questions

If you have any further questions or queries, feel free to raise an issue on GitHub. Else, come join our [Discord](https://hatchet.run/discord)!
