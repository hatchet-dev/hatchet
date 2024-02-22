[![Docs](https://img.shields.io/badge/docs-docs.hatchet.run-3F16E4)](https://docs.hatchet.run) [![Discord](https://img.shields.io/discord/1088927970518909068?style=social&logo=discord)](https://discord.gg/ZMeUafwH89) [![License: MIT](https://img.shields.io/badge/License-MIT-purple.svg)](https://opensource.org/licenses/MIT) [![Go Reference](https://pkg.go.dev/badge/github.com/hatchet-dev/hatchet.svg)](https://pkg.go.dev/github.com/hatchet-dev/hatchet)

## Introduction

Hatchet is a self-hostable platform which lets you define and scale workflows as code.

**You run your workers, Hatchet manages the rest.**

Hatchet is an orchestrator, which means it manages the execution of your workflows. The individual steps of each workflow are executed by your own workers (don't worry, each SDK comes with a worker implementation). This means you can run your workers in your own infrastructure, and Hatchet will manage the scheduling, retries, and monitoring of your workflows. We then provide a full observability layer and dashboard for debugging and retrying failed executions, along with an API for programmatically managing workflows.

## Use-Cases

Hatchet is a generalized, low-latency workflow engine designed for application developers, but is particularly useful in the following cases:

### Background Task Management and Scheduling

Instead of developers interfacing directly with a task queue, Hatchet provides a simple API built into each SDK for managing background tasks.

**Retries, timeouts and error handling** are built into each Hatchet SDK.

<!-- <IMAGE> -->

- **Cron schedules and scheduled workflows** schedule workflows using a crontab syntax, like `*/15 * * * *` (every 15 minutes). You can set multiple crons per workflows, or schedule one-off workflows in the future.

- **Task observability** with Hatchet, you get complete access to the inputs and outputs from each step run, which is useful for debugging and observability.

### Prompt Engineering Playground

Hatchet lets you expose the existing methods you've built in your LLM-enabled applications on a UI for better observability and prompt iteration. It looks something like this:

<!-- <GIF> -->

- **UI-based iteration of LLM workflows** - you get full flexibility to choose which variables to expose on the playground. We do this by providing a method in our SDK called `playground` which then exposes the variable in the Hatchet UI:

<!-- <GIF> -->

- **Full observability into customer interactions** with Hatchet, you automatically get a full history of the inputs and outputs to each step in your workflow, which is particularly useful when debugging bad customer interactions with your LLMs.

<!-- <GIF> -->

- **Deploy changes to Github** useful for non-technical founders and product managers to quickly request changes to your codebase without waiting for an engineer.

<!-- <GIF> -->

### Event-Driven Architectures

- **Durable event log** - get a full history of events within your system that triggered workflows, with an Events API for pushing and pulling events.

- **Logically organize your services** - each worker can run its own set of workflows, so you can organize your worker pools to only pickup certain types of tasks.

## Getting Started

To get started, see the Hatchet documentation [here](https://docs.hatchet.run/home/quickstart), or check out our quickstart repos:

- [Go SDK Quickstart](https://github.com/hatchet-dev/hatchet-go-quickstart)
- [Python SDK Quickstart](https://github.com/hatchet-dev/hatchet-python-quickstart)

## Issues

Please submit any bugs that you encounter via Github issues. However, please reach out on [Discord](https://discord.gg/ZMeUafwH89) before submitting a feature request - as the project is very early, we'd like to build a solid foundation before adding more complex features.

## I'd Like to Contribute

See the contributing docs [here](https://docs.hatchet.run/contributing), and please let us know what you're interesting in working on in the #contributing channel on [Discord](https://discord.gg/ZMeUafwH89). This will help us shape the direction of the project and will make collaboration much easier!
