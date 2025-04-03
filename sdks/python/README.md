# Hatchet Python SDK

<div align="center">

[![PyPI version](https://badge.fury.io/py/hatchet-sdk.svg)](https://badge.fury.io/py/hatchet-sdk)
[![Documentation](https://img.shields.io/badge/docs-hatchet.run-blue)](https://docs.hatchet.run)
[![License: MIT](https://img.shields.io/badge/License-MIT-purple.svg)](https://opensource.org/licenses/MIT)

</div>

This is the official Python SDK for [Hatchet](https://hatchet.run), a distributed, fault-tolerant task queue. The SDK allows you to easily integrate Hatchet's task scheduling and workflow orchestration capabilities into your Python applications.

## Installation

Install the SDK using pip:

```bash
pip install hatchet-sdk
```

Or using poetry:

```bash
poetry add hatchet-sdk
```

## Quick Start

For examples of how to use the Hatchet Python SDK, including worker setup and task execution, please see our [official documentation](https://docs.hatchet.run/home/setup).

## Features

- 🔄 **Workflow Orchestration**: Define complex workflows with dependencies and parallel execution
- 🔁 **Automatic Retries**: Configure retry policies for handling transient failures
- 📊 **Observability**: Track workflow progress and monitor execution metrics
- ⏰ **Scheduling**: Schedule workflows to run at specific times or on a recurring basis
- 🔄 **Event-Driven**: Trigger workflows based on events in your system

## Documentation

For detailed documentation, examples, and best practices, visit:
- [Hatchet Documentation](https://docs.hatchet.run)
- [Examples](https://github.com/hatchet-dev/hatchet/tree/main/sdks/python/examples)

## Contributing

We welcome contributions! Please check out our [contributing guidelines](https://docs.hatchet.run/contributing) and join our [Discord community](https://hatchet.run/discord) for discussions and support.

## License

This SDK is released under the MIT License. See [LICENSE](https://github.com/hatchet-dev/hatchet/blob/main/LICENSE) for details.
