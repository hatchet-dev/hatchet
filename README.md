[![Discord](https://img.shields.io/discord/1088927970518909068?style=social&logo=discord)](https://discord.gg/ZMeUafwH89) [![License: MIT](https://img.shields.io/badge/License-MIT-purple.svg)](https://opensource.org/licenses/MIT) [![Go Reference](https://pkg.go.dev/badge/github.com/hatchet-dev/hatchet.svg)](https://pkg.go.dev/github.com/hatchet-dev/hatchet)

## Introduction

_**Note:** Hatchet is in early development. Changes are not guaranteed to be backwards-compatible. If you'd like to run Hatchet in production, feel free to reach out on Discord for tips._

Hatchet is a self-hostable workflow engine built for application developers.

**What is a workflow?**

The term `workflow` tends to be overloaded, so let's make things more clear - in Hatchet, a workflow is a set of functions which are executed in response to an external trigger (an event, schedule, or API call). For example, if you'd like to send notifications to a user after they've signed up, you could create a workflow for that.

**Why is that useful?**

Instead of processing background tasks and functions in your application handlers, which can lead to complex code, hard-to-debug errors, and resource contention, you can distribute these workflows between a set of `workers`. Workers are long-running processes which listen for events, and execute the functions defined in your workflows.

**What is a workflow engine?**

A workflow engine orchestrates the execution of workflows. It schedules workflows on workers, retries failed workflows, and provides integrations for monitoring and debugging workflows.

## Project Goals

Hatchet has the following high-level goals:

1. **Serve application developers:** we aim to support a broad set of languages and frameworks, to make it easier to support your existing applications. We currently support a Go SDK, with more languages coming soon.
2. **Simple to setup:** we've seen too many overengineered stacks built on a fragile task queue with overly complex infrastructure. Hatchet is designed to be simple to setup, run locally, and deploy to your own infrastructure.
3. **Flexibility when you need it:** as your application grows, you can use Hatchet to support complex, multi-step distributed workflows. Hatchet's backend is modular, allowing for customizing the implementation of the event storage API, queueing system, authentication, and more.

## Features

**Currently implemented**

- **✅ Declarative workflows:** use the Go SDK to define workflows, with support for timeouts and parallel execution.
- **✅ Cron schedules:** schedule workflows using a crontab syntax, like `*/15 * * * *` (every 15 minutes).
- **✅ Events API**: store events in a durable event log, with support for querying and filtering events. Define which events trigger which workflows.
- **✅ Web UI**: use the web UI to monitor and debug your workflows and events.
- **✅ Self-hostable**: MIT-licensed and Docker images available.
- **✅ Locally runnable**: see [here](https://github.com/hatchet-dev/hatchet-go-quickstart) for an example.
- **✅ Organize workflows using services**: use `worker.NewService` to organize your workflows.

**Near-term roadmap**

- 🚧 Helm chart for Kubernetes deployments
- 🚧 UI and CLI for creating, updating, and deleting workflows
- 🚧 Better support for parallel step execution
- 🚧 More integrations

## Getting Started

To get started, see the Hatchet documentation [here](https://docs.hatchet.run).

## Github Issues

Please submit any bugs that you encounter via Github issues. However, please reach out on [Discord](https://discord.gg/ZMeUafwH89) before submitting a feature request - as the project is very early, we'd like to build a solid foundation before adding more complex features.

## I'd Like to Contribute

See the contributing docs [here](./CONTRIBUTING.md), and please let us know what you're interesting in working on in the #contributing channel on [Discord](https://discord.gg/ZMeUafwH89). This will help us shape the direction of the project and will make collaboration much easier!
