
<div align="center">

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="https://framerusercontent.com/images/KBMnpSO12CyE6UANhf4mhrg6na0.png?scale-down-to=200">
  <source media="(prefers-color-scheme: light)" srcset="https://framerusercontent.com/images/KBMnpSO12CyE6UANhf4mhrg6na0.png?scale-down-to=200">
  <a href ="https://hatchet.run">
	  <img alt="Hatchet Logo" src="https://framerusercontent.com/images/KBMnpSO12CyE6UANhf4mhrg6na0.png?scale-down-to=200">
  </a>
</picture>

### The open source low-latency queue for web apps at scale

[![Docs](https://img.shields.io/badge/docs-docs.hatchet.run-3F16E4)](https://docs.hatchet.run) [![Discord](https://img.shields.io/discord/1088927970518909068?style=social&logo=discord)](https://discord.gg/ZMeUafwH89) [![License: MIT](https://img.shields.io/badge/License-MIT-purple.svg)](https://opensource.org/licenses/MIT) [![Go Reference](https://pkg.go.dev/badge/github.com/hatchet-dev/hatchet.svg)](https://pkg.go.dev/github.com/hatchet-dev/hatchet) [![NPM Downloads](https://img.shields.io/npm/dm/%40hatchet-dev%2Ftypescript-sdk)](https://www.npmjs.com/package/@hatchet-dev/typescript-sdk)
[![Twitter](https://img.shields.io/twitter/url/https/twitter.com/hatchet-dev.svg?style=social&label=Follow%20%40hatchet-dev)](https://twitter.com/triggerdotdev) [![GitHub Repo stars](https://img.shields.io/github/stars/hatchet-dev/hatchet?style=social)](https://github.com/triggerdotdev/trigger.dev)

</div>
## # What is Hatchet?

Hatchet is a managed or self-hosted low-latency queue for your web apps to solve problems at scale like concurrency, fairness, and rate limiting. Hatchet is used wherever handling requests due to scaling is an issue. For example, you might use hatchet for reliably handling high volume generative AI requests, orchestrate durable data workflows, schedule large-scale batch jobs, and much more.

Instead of processing background tasks and functions in your application handlers, which can lead to complex code, hard-to-debug errors, and resource contention, you can distribute these workflows between a set of workers. Workers are long-running processes which listen for events, and execute the functions defined in your workflows

#### Is Hatchet a workflow engine?

Hatchet is designed to be a simple, reliable, and scalable way to handle background tasks and functions in your web application. While Hatchet supports full-featured and declarative DAG workflows, it's low latency design makes it a great fit for a wide range of use cases including servicing real-time requests that might be only a single function.

**What Makes Hatchet Great?**
- ⚡️ **Ultra-low Latency and High Throughput Scheduling:** Hatchet stands as the fastest workflow engine, perfectly balancing real-time interaction capabilities with the reliability required for mission-critical tasks. [Benchmarks →](https://docs.hatchet.run)

- ☮️ **Concurrency, Fairness, and Rate Limiting:** Implement FIFO, LIFO, Round Robin, and Priority Queues effortlessly with Hatchet’s built-in strategies, designed to circumvent common scaling pitfalls with minimal configuration. [Read Docs →](https://docs.hatchet.run)

- 🔥🧯 **Resilience by Design:** With customizable retry policies and integrated error handling, Hatchet ensures your operations recover swiftly from transient failures. You can break large jobs down into small tasks so you can finish a run without rerunning work. [Read Docs →](https://docs.hatchet.run)

**Enhanced Visibility and Control:**
- **Observability.** All of your runs are fully searchable, allowing you to quickly identify issues. We track latency, error rates, or custom metrics in your run.
- **(Practical) Durable Execution.** Replay events and manually pick up execution from specific steps in your workflow.
- **Cron.** Set recurring schedules for functions runs to execute.
- **One-Time Scheduling.** Schedule a function run to execute at a specific time and date in the future.
- **Spike Protection.** Smooth out spikes in traffic and only execute what your system can handle.
- **Incremental Streaming.** Subscribe to updates as your functions progress in the background worker.

**Example Use Cases:**
- **Fairness for Generative AI:** Ensure equitable distribution of requests across your system with Hatchet’s configurable policies, preventing system overload by busy users
- **Batch Processing for Document Indexing:** Effortlessly manage large-scale batch processing tasks, with the ability to pause and resume operations seamlessly in case of failures.
- **Workflow Orchestration for Multi-Modal Systems:** Expertly coordinate workflows involving multi-modal inputs and outputs, supported by comprehensive DAG-style execution.
- **Correctness for Event-Based Processing:** Automatically respond and replay events, enhancing system reliability and data integrity.

## Quick Start
Hatchet supports your technology stack with open-source SDKs for Python, Typescript, and Go, allowing for declarative function definition and offering the flexibility to adapt to emerging technologies.

To get started, see the Hatchet documentation [here](https://docs.hatchet.run/home/quickstart), or check out our quickstart repos:
- [Go SDK Quickstart](https://github.com/hatchet-dev/hatchet-go-quickstart)
- [Python SDK Quickstart](https://github.com/hatchet-dev/hatchet-python-quickstart)
- [Typescript SDK Quickstart](https://github.com/hatchet-dev/hatchet-typescript-quickstart) (coming soon!)
 
#### Is there a managed cloud version of Hatchet?

Yes, we are offering a have a cloud version to select companies while in beta who are helping to build and shape the product. Please [reach out](mailto:contact@hatchet.run) or [request access](https://hatchet.run/request-access) for more information.

#### Is there a self-hosted version of Hatchet?

Yes, instructions for self-hosting our open source docker containers can be found in our  [documentation](https://docs.hatchet.run/self-hosting/docker-compose). Please [reach out](mailto:contact@hatchet.run)  if you're interested in support.

## Issues
Please submit any bugs that you encounter via Github issues. However, please reach out on [Discord](https://discord.gg/ZMeUafwH89) before submitting a feature request - as the project is very early, we'd like to build a solid foundation before adding more complex features.

## I'd Like to Contribute

See the contributing docs [here](https://docs.hatchet.run/contributing), and please let us know what you're interesting in working on in the #contributing channel on [Discord](https://discord.gg/ZMeUafwH89). This will help us shape the direction of the project and will make collaboration much easier!