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

## Getting Started

To get started, see the Hatchet documentation [here](https://docs.hatchet.run).

## Github Issues

Please submit any bugs that you encounter via Github issues. However, please reach out on [Discord](https://discord.gg/ZMeUafwH89) before submitting a feature request - as the project is very early, we'd like to build a solid foundation before adding more complex features.

## I'd Like to Contribute

See the contributing docs [here](./CONTRIBUTING.md), and please let us know what you're interesting in working on in the #contributing channel on [Discord](https://discord.gg/ZMeUafwH89). This will help us shape the direction of the project and will make collaboration much easier!
