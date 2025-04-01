<div align="center">

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="https://framerusercontent.com/images/KBMnpSO12CyE6UANhf4mhrg6na0.png?scale-down-to=200">
  <source media="(prefers-color-scheme: light)" srcset="https://framerusercontent.com/images/KBMnpSO12CyE6UANhf4mhrg6na0.png?scale-down-to=200">
  <a href ="https://hatchet.run">
	  <img alt="Hatchet Logo" src="https://framerusercontent.com/images/KBMnpSO12CyE6UANhf4mhrg6na0.png?scale-down-to=200">
  </a>
</picture>

### A Distributed, Fault-Tolerant Task Queue

[![Docs](https://img.shields.io/badge/docs-docs.hatchet.run-3F16E4)](https://docs.hatchet.run) [![License: MIT](https://img.shields.io/badge/License-MIT-purple.svg)](https://opensource.org/licenses/MIT) [![Go Reference](https://pkg.go.dev/badge/github.com/hatchet-dev/hatchet.svg)](https://pkg.go.dev/github.com/hatchet-dev/hatchet) [![NPM Downloads](https://img.shields.io/npm/dm/%40hatchet-dev%2Ftypescript-sdk)](https://www.npmjs.com/package/@hatchet-dev/typescript-sdk)

[![Discord](https://img.shields.io/discord/1088927970518909068?style=social&logo=discord)](https://hatchet.run/discord)
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

Hatchet replaces difficult to manage legacy queues or pub/sub systems so you can design durable workloads that recover from failure and solve for problems like **concurrency**, **fairness**, and **rate limiting**. Instead of managing your own task queue or pub/sub system, you can use Hatchet to distribute your functions between a set of workers with minimal configuration or infrastructure:

<p align="center">
  <img width="500" height="500" src="https://github.com/hatchet-dev/hatchet/assets/25448214/c3defa1e-d9d9-4419-94e5-b4ea4a748f8d">
</p>

**What Makes Hatchet Great?**

- ⚡️ **Ultra-low Latency and High Throughput Scheduling:** Hatchet is built on a low-latency queue, perfectly balancing real-time interaction capabilities with the reliability required for mission-critical tasks.

- ☮️ **Concurrency, Fairness, and Rate Limiting:** Implement FIFO, LIFO, Round Robin, and Priority Queues with Hatchet’s built-in strategies, designed to circumvent common scaling pitfalls with minimal configuration. [Read Docs →](https://docs.hatchet.run)

- 🔥🧯 **Resilience by Design:** With customizable retry policies and integrated error handling, Hatchet ensures your operations recover swiftly from transient failures. You can break large jobs down into small tasks so you can finish a run without rerunning work. [Read Docs →](https://docs.hatchet.run)

**Enhanced Visibility and Control:**

- **Observability.** All of your runs are fully searchable, allowing you to quickly identify issues. We track latency, error rates, or custom metrics in your run.
- **(Practical) Durable Execution.** Replay events and manually pick up execution from specific steps in your workflow.
- **Cron.** Set recurring schedules for functions runs to execute.
- **One-Time Scheduling.** Schedule a function run to execute at a specific time and date in the future.
- **Spike Protection.** Smooth out spikes in traffic and only execute what your system can handle.
- **Incremental Streaming.** Subscribe to updates as your functions progress in the background worker.

**Example Use Cases:**

- **AI Agents:** define your agentic workflows as code and leverage Hatchet to retry failures and parallelize agent actions.
- **Fairness for Generative AI:** Don't let busy users overwhelm your system. Hatchet lets you distribute requests to your workers fairly with configurable policies.
- **Batch Processing for Document Indexing:** Hatchet can handle large-scale batch processing of documents, images, and other data and resume mid-job on failure.
- **Workflow Orchestration for Multi-Modal Systems:** Hatchet can handle orchestrating multi-modal inputs and outputs, with full DAG-style execution.
- **Correctness for Event-Based Processing:** Respond to external events or internal events within your system and replay events automatically.

## Quick Start

Hatchet is available as a cloud version or self-hosted. See the following docs to get up and running quickly:

- [Hatchet Cloud Quickstart](https://docs.hatchet.run/home/hatchet-cloud-quickstart)
- [Hatchet Self-Hosted](https://docs.hatchet.run/self-hosting)

Hatchet supports your technology stack with open-source SDKs for Python, Typescript, and Go. To get started, see the [quickstart guide](https://docs.hatchet.run/home/setup).

### SDKs

If you encounter any issues while using the SDKs, please [submit an issue](https://github.com/hatchet-dev/hatchet/issues):

## How does this compare to alternatives (Celery, BullMQ)?

Why build another managed queue? We wanted to build something with the benefits of full transactional enqueueing - particularly for dependent, DAG-style execution - and felt strongly that Postgres solves for 99.9% of queueing use-cases better than most alternatives (Celery uses Redis or RabbitMQ as a broker, BullMQ uses Redis). Since the introduction of `SKIP LOCKED` and the milestones of recent PG releases (like active-active replication), it's becoming more feasible to horizontally scale Postgres across multiple regions and vertically scale to 10k TPS or more. Many queues (like BullMQ) are built on Redis and data loss can occur when suffering OOM if you're not careful, and using PG helps avoid an entire class of problems.

We also wanted something that was significantly easier to use and debug for application developers. A lot of times the burden of building task observability falls on the infra/platform team (for example, asking the infra team to build a Grafana view for their tasks based on exported prom metrics). We're building this type of observability directly into Hatchet.

For more information for why we built Hatchet, you can check out our writeup on Celery [here](https://docs.hatchet.run/blog/problems-with-celery).

## Issues

Please submit any bugs that you encounter via Github issues. However, please reach out on [Discord](https://hatchet.run/discord) before submitting a feature request - as the project is very early, we'd like to build a solid foundation before adding more complex features.

## I'd Like to Contribute

See the contributing docs [here](https://docs.hatchet.run/contributing), and please let us know what you're interesting in working on in the #contributing channel on [Discord](https://hatchet.run/discord). This will help us shape the direction of the project and will make collaboration much easier!
