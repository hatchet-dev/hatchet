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

[![Discord](https://img.shields.io/discord/1088927970518909068?style=social&logo=discord)](https://discord.gg/ZMeUafwH89)
[![Twitter](https://img.shields.io/twitter/url/https/twitter.com/hatchet-dev.svg?style=social&label=Follow%20%40hatchet-dev)](https://twitter.com/hatchet_dev)
[![GitHub Repo stars](https://img.shields.io/github/stars/hatchet-dev/hatchet?style=social)](https://github.com/hatchet-dev/hatchet)

  <p align="center">
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

- ⚡️ **Ultra-low Latency and High Throughput Scheduling:** Hatchet is built on a low-latency queue (`25ms` average start), perfectly balancing real-time interaction capabilities with the reliability required for mission-critical tasks.

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

- **Fairness for Generative AI:** Don't let busy users overwhelm your system. Hatchet lets you distribute requests to your workers fairly with configurable policies.
- **Batch Processing for Document Indexing:** Hatchet can handle large-scale batch processing of documents, images, and other data and resume mid-job on failure.
- **Workflow Orchestration for Multi-Modal Systems:** Hatchet can handle orchestrating multi-modal inputs and outputs, with full DAG-style execution.
- **Correctness for Event-Based Processing:** Respond to external events or internal events within your system and replay events automatically.

## Quick Start

Hatchet supports your technology stack with open-source SDKs for Python, Typescript, and Go. To get started, see the Hatchet documentation [here](https://docs.hatchet.run/home/quickstart), or check out our quickstart repos:

- [Go SDK Quickstart](https://github.com/hatchet-dev/hatchet-go-quickstart)
- [Python SDK Quickstart](https://github.com/hatchet-dev/hatchet-python-quickstart)
- [Typescript SDK Quickstart](https://github.com/hatchet-dev/hatchet-typescript-quickstart)

#### Is there a managed cloud version of Hatchet?

Yes, we are offering a have a cloud version to select companies while in beta who are helping to build and shape the product. Please [reach out](mailto:contact@hatchet.run) or [request access](https://hatchet.run/request-access) for more information.

#### Is there a self-hosted version of Hatchet?

Yes, instructions for self-hosting our open source docker containers can be found in our [documentation](https://docs.hatchet.run/self-hosting/docker-compose). Please [reach out](mailto:contact@hatchet.run) if you're interested in support.

## Issues

Please submit any bugs that you encounter via Github issues. However, please reach out on [Discord](https://discord.gg/ZMeUafwH89) before submitting a feature request - as the project is very early, we'd like to build a solid foundation before adding more complex features.

## I'd Like to Contribute

See the contributing docs [here](https://docs.hatchet.run/contributing), and please let us know what you're interesting in working on in the #contributing channel on [Discord](https://discord.gg/ZMeUafwH89). This will help us shape the direction of the project and will make collaboration much easier!
