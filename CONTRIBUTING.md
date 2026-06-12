# Contributing

This guide will help you understand how to contribute effectively to the Hatchet project.


## Guidelines

The following requirements apply to all contributions.

- First-time contributors may have at most one open pull request at a time.
- Issues labeled [![good first issue](https://img.shields.io/github/labels/hatchet-dev/hatchet/good%20first%20issue)](https://github.com/hatchet-dev/hatchet/issues?q=is%3Aissue%20state%3Aopen%20label%3A%22good%20first%20issue%22) are reserved for first-time contributors.
- Pull requests must reference a corresponding issue labeled [![accepted](https://img.shields.io/github/labels/hatchet-dev/hatchet/accepted)](https://github.com/hatchet-dev/hatchet/issues?q=is%3Aissue%20state%3Aopen%20label%3Aaccepted).
- Your GitHub account's [Activity Overview](https://docs.github.com/en/account-and-profile/how-tos/contribution-settings/showing-an-overview-of-your-activity-on-your-profile) must be public.
- AI usage must be disclosed and comply with [AI_POLICY.md](./AI_POLICY.md) (see [AI Usage](#ai-usage)).

### AI Usage

Pull requests, issues, and discussions that use AI require explicit disclosure. For example:

> <details open id="ai-disclosure">
> <summary><b>🤖 AI Disclosure</b></summary>
>
> <!-- In accordance with Hatchet's AI_POLICY.md, LLM usage must be explicitly disclosed. -->
>
> - [x] _I acknowledge that an LLM was used in the creation of this Pull Request, in accordance with Hatchet's [AI_POLICY.md](./AI_POLICY.md)._
>
> <!-- Please specify the tooling/model and the extent to which it was used. -->
>
> - **Details**: Claude Code was used to generate the TypeScript SDK tests.
>
> </details>


## Getting Started

New to Hatchet? Start with our [Architecture](https://docs.hatchet.run/home/architecture) docs to familiarize yourself with Hatchet's core system design.

Then, before contributing, check out the following sections:

- [Development Environment Setup](#development-environment-setup)
- [Pull Requests](#pull-requests)
- [Testing](#testing)
- [Running Locally](#running-locally)
   - [Example Workflow](#example-workflow)

## Development Environment Setup

Ensure all prerequisite dependencies are installed:

- [Go 1.26+](https://go.dev/doc/install)
- [Node.js v18+](https://nodejs.org/en/download)
   - We recommend using [nvm](https://github.com/nvm-sh/nvm) for managing node versions to match the version defined in [`.nvmrc`](.nvmrc)
- [pnpm](https://pnpm.io/installation) installed globally (`npm i -g pnpm`)
- [Docker](https://docs.docker.com/engine/install/)
- [task](https://taskfile.dev/docs/installation)
- [protoc](https://grpc.io/docs/protoc-installation/)
- [Caddy](https://caddyserver.com/docs/install)
- [goose](https://pressly.github.io/goose/installation/)
- [pre-commit](https://pre-commit.com/)
   - You can install this in a virtual environment with `task pre-commit-install`

We recommend installing these tools individually using your preferred package manager (e.g., Homebrew).

## Pull Requests

Before opening a PR, check if there's a related and accepted issue in our [backlog](https://github.com/hatchet-dev/hatchet/issues).

For non-trivial changes (anything beyond typos or patch version bumps), please create an issue first so we can discuss the proposal and ensure it aligns with the project.

Next, ensure all changes are:

- Unit tested with `task test`
- Linted with `task lint`
- Formatted with `task fmt`
- Integration tested with `task test-integration` (when applicable)

If your changes require documentation updates, modify the relevant files in [`frontend/docs/pages/`](frontend/docs/pages/). You can spin up the documentation site locally by running `task docs`. By default, this will be available at [`http://localhost:3000`](http://localhost:3000).

For configuration changes, see [Updating Configuration](contributing/developer-guides/updating-configuration.md).

### Guidelines

Pull request titles should be conform to the [conventional commit](https://www.conventionalcommits.org/) format i.e

```
<type>(<scope>): <short description>
```

#### Scope

Pull request titles can be (optionally) scoped to specify the affected area of the codebase. If multiple scopes apply, they can be provided as a comma-delimited list, e.g. `feat(sdks/go,sdks/ts): ...`. An empty scope implies the change is cross-cutting or not changelog-relevant, e.g. `chore: fix typo in README`.

Please use the following when scoping your changes:

**Hatchet core:**
- `engine`
- `api`
- `migrate`
- `admin`
- `cli`
- `dashboard`
- `lite`

**Hatchet SDKs:**
- `sdks/python`
- `sdks/ruby`
- `sdks/go`
- `sdks/ts`

**Other:**
- `ci`
- `docs`
- `devex`

> [!NOTE]
> Future tooling will rely on scoping to disambiguate the surface area of changes, so please scope your PR where applicable. This list is subject to change.

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
