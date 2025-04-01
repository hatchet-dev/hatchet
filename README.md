<div align="center">

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="https://framerusercontent.com/images/KBMnpSO12CyE6UANhf4mhrg6na0.png?scale-down-to=200">
  <source media="(prefers-color-scheme: light)" srcset="https://framerusercontent.com/images/KBMnpSO12CyE6UANhf4mhrg6na0.png?scale-down-to=200">
  <a href ="https://hatchet.run">
	  <img alt="Hatchet Logo" src="https://framerusercontent.com/images/KBMnpSO12CyE6UANhf4mhrg6na0.png?scale-down-to=200">
  </a>
</picture>

### Run Background Tasks at Scale

[![Docs](https://img.shields.io/badge/docs-docs.hatchet.run-3F16E4)](https://docs.hatchet.run) [![License: MIT](https://img.shields.io/badge/License-MIT-purple.svg)](https://opensource.org/licenses/MIT) [![Go Reference](https://pkg.go.dev/badge/github.com/hatchet-dev/hatchet.svg)](https://pkg.go.dev/github.com/hatchet-dev/hatchet) [![NPM Downloads](https://img.shields.io/npm/dm/%40hatchet-dev%2Ftypescript-sdk)](https://www.npmjs.com/package/@hatchet-dev/typescript-sdk)

[![Discord](https://img.shields.io/discord/1088927970518909068?style=social&logo=discord)](https://discord.gg/ZMeUafwH89)
[![Twitter](https://img.shields.io/twitter/url/https/twitter.com/hatchet-dev.svg?style=social&label=Follow%20%40hatchet-dev)](https://twitter.com/hatchet_dev)
[![GitHub Repo stars](https://img.shields.io/github/stars/hatchet-dev/hatchet?style=social)](https://github.com/hatchet-dev/hatchet)

  <p align="center">
    <a href="https://cloud.onhatchet.run">Hatchet Cloud</a>
    ·
    <a href="https://docs.hatchet.run">Documentation</a>
    ·
    <a href="https://hatchet.run">Website</a>
    ·
    <a href="https://github.com/hatchet-dev/hatchet/issues">Issues</a>
  </p>

</div>

### What is Hatchet?

Hatchet is a platform for running background tasks, built on top of Postgres. Instead of managing your own task queue or pub/sub system, you can use Hatchet to distribute your functions between a set of workers with minimal configuration or infrastructure -- the only required dependency is Postgres.

It includes the following features:

<details>

<summary>Durable task queue</summary>

Hatchet is a durable task queue -- tasks are enqueued by calling an endpoint from the Hatchet SDKs and run on **workers** that you manage. Hatchet will track the progress of your task and ensure that the work gets completed (or you get alerted), even if your application crashes.

This is particularly useful for:

- Ensuring that you never drop a user request
- Flattening large spikes in your application
- Breaking large, complex logic into smaller, reusable tasks

</details>

<details>

<summary>Flow control</summary>

Hatchet allows you to throttle execution on a per-user, per-tenant and per-queue basis, increasing system stability and limiting the impact of busy users on the rest of your system.

Hatchet supports the following flow control primitives:

- Concurrency — you can set a concurrency limit based on a dynamic concurrency key (e.g., each user can only run 10 batch jobs at a given time).
- Rate limiting — you can create both global and dynamic rate limits.

</details>

<details>

<summary>Task orchestration</summary>

Hatchet allows you to build complex workflows that can be composed of multiple tasks. For example, if you'd like to break a workload into smaller tasks, you can use Hatchet to create a fanout workflow that spawns multiple tasks in parallel.

Hatchet supports the following mechanisms for task orchestration:

- DAGs (directed acyclic graphs) — pre-define the shape of your work, automatically routing the outputs of a parent task to the input of a child task. Read more ➶
- Durable tasks — these tasks are responsible for orchestrating other tasks. They store a full history of all spawned tasks, allowing you to cache intermediate results. Read more ➶

</details>

<details>

<summary>Scheduling</summary>

Hatchet has full support for cron, one-time scheduling, and pausing execution for a time duration. This is particularly useful for:

- Handling batch processes on a cron schedule
- Processing a task at a specific time
- Waiting for a duration until resuming task execution

</details>

<details>

<summary>Task Routing</summary>

While the default Hatchet behavior is to implement a FIFO queue, it also supports additional scheduling mechanisms to route your tasks to the ideal worker:

- Sticky assignment — allows spawned tasks to prefer execution on the same worker.
- Worker affinity — ranks workers to discover which is best suited to handle a given task.

</details>

<details>

<summary>Event Signaling</summary>

Hatchet supports event-based architectures where tasks and workflows can pause execution while waiting for a specific external event. Events can be filtered using common expression language, enabling more reactive applications. They can also trigger new workflows and tasks.

</details>

<details>

<summary>Web Dashboard</summary>

Hatchet bundles a web dashboard for real-time querying of tasks and workflows:

TODO

</details>

<details>

<summary>Built-in Monitoring and Alerting</summary>

Hatchet has built-in support for configuring alerts (using Slack or email) for task and workflow failures and logging from workflows:

TODO

</details>

### Documentation

The most up-to-date documentation can be found at https://docs.hatchet.run.

### Community & Support

- [Discord](https://discord.gg/ZMeUafwH89) - best for getting in touch with the maintainers and hanging with the community
- [Github Issues](https://github.com/hatchet-dev/hatchet/issues) - used for filing bug reports
- [Github Discussions](https://github.com/hatchet-dev/hatchet/discussions) - used for starting in-depth technical discussions that are suited for asynchronous communication
- [Email](mailto:contact@hatchet.run) - best for getting Hatchet Cloud support and for help with billing, data deletion, etc.

### Example Use Cases

- **AI Agents:** define your agentic workflows as code and leverage Hatchet to retry failures and parallelize agent actions.
- **Fairness for Generative AI:** Don't let busy users overwhelm your system. Hatchet lets you distribute requests to your workers fairly with configurable policies.
- **Batch Processing for Document Indexing:** Hatchet can handle large-scale batch processing of documents, images, and other data and resume mid-job on failure.
- **Workflow Orchestration for Multi-Modal Systems:** Hatchet can handle orchestrating multi-modal inputs and outputs, with full DAG-style execution.
- **Correctness for Event-Based Processing:** Respond to external events or internal events within your system and replay events automatically.

### Quick Start

Hatchet is available as a cloud version or self-hosted. See the following docs to get up and running quickly:

- [Hatchet Cloud Quickstart](https://docs.hatchet.run/home/hatchet-cloud-quickstart)
- [Hatchet Self-Hosted](https://docs.hatchet.run/self-hosting)

Hatchet supports your technology stack with open-source SDKs for Python, Typescript, and Go. To get started, see the [quickstart guide](https://docs.hatchet.run/home/setup).

### SDKs

If you encounter any issues while using the SDKs, please [submit an issue](https://github.com/hatchet-dev/hatchet/issues):

### Hatchet vs...

#### Hatchet vs Temporal

Hatchet is designed to be a general-purpose task orchestration platform -- it can be used as a queue, a DAG-based orchestrator, a durable execution engine, or all three. As a result, Hatchet covers a wider array of use-cases, like multiple queueing strategies, rate limiting, DAG features like conditional triggering, streaming features, and much more.

Temporal is narrowly focused on the durable execution pattern, but supports a wider range of database backends and result stores, like Apache Cassandra, MySQL, PostgreSQL, and SQLite.

**When to use Hatchet:** when you'd like to get more control over the underlying queue, run DAG-based workflows, or want to simplify self-hosting by only running the Hatchet engine and Postgres.

**When to use Temporal:** when you'd like to use a non-Postgres result store, or your only workload is best suited for durable execution.

#### Hatchet vs Task Queues (BullMQ, Celery)

Hatchet is a durable task queue, meaning it persists the history of all executions (up to a retention period), which allows for easy monitoring + debugging and powers a bunch of the durability features above. This isn’t the standard behavior of Celery and BullMQ (and you need to rely on third-party UI tools which are extremely limited in functionality, like Celery Flower).

**When to use Hatchet:** when you'd like to results to be persisted and observable in a UI

**When to use task queue library like BullMQ/Celery:** when you'd like to use a single library (instead of a standalone service) to interact with your queue

#### Hatchet vs DAG-based platforms (Airflow, Prefect, Dagster)

These tools are usually built with data engineers in mind, and aren’t meant to run from a high-volume application. They’re usually lower latency and higher cost, with their primary selling point being integrations with common datastores and connectors.

#### Hatchet vs AI Frameworks

Most AI frameworks are built to run in-memory, with horizontal scaling and durability as an afterthought. While you can use an AI framework in conjunction with Hatchet, most of our users discard their AI framework and use Hatchet’s primitives to build their applications.

### Issues

Please submit any bugs that you encounter via Github issues. However, please reach out on [Discord](https://discord.gg/ZMeUafwH89) before submitting a feature request - as the project is very early, we'd like to build a solid foundation before adding more complex features.

### I'd Like to Contribute

Please let us know what you're interesting in working on in the #contributing channel on [Discord](https://discord.gg/ZMeUafwH89). This will help us shape the direction of the project and will make collaboration much easier!
