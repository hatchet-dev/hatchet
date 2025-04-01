# Hatchet TypeScript SDK

<div align="center">

[![npm version](https://badge.fury.io/js/@hatchet-dev%2Ftypescript-sdk.svg)](https://badge.fury.io/js/@hatchet-dev%2Ftypescript-sdk)
[![Documentation](https://img.shields.io/badge/docs-hatchet.run-blue)](https://docs.hatchet.run)
[![License: MIT](https://img.shields.io/badge/License-MIT-purple.svg)](https://opensource.org/licenses/MIT)

</div>

This is the official TypeScript SDK for [Hatchet](https://hatchet.run), a distributed, fault-tolerant task queue. The SDK provides a type-safe way to integrate Hatchet's task scheduling and workflow orchestration capabilities into your TypeScript/JavaScript applications.

## Installation

Install the SDK using npm:

```bash
npm install @hatchet-dev/typescript-sdk
```

Using yarn:

```bash
yarn add @hatchet-dev/typescript-sdk
```

Using pnpm:

```bash
pnpm add @hatchet-dev/typescript-sdk
```

## Quick Start

Here's a simple example of how to use the Hatchet TypeScript SDK:

```typescript
import { HatchetClient } from '@hatchet-dev/typescript-sdk';

export const hatchet = HatchetClient.init();

export type SimpleInput = {
  Message: string;
};

export const simple = hatchet.task({
  name: 'simple',
  fn: (input: SimpleInput) => {
    return {
      TransformedMessage: input.Message.toLowerCase(),
    };
  },
});

async function main() {
  const worker = await hatchet.worker('simple-worker', {
    workflows: [simple],
    slots: 100,
  });

  await worker.start();
}

if (require.main === module) {
  main();
}
```

## Features

- 📝 **Type Safety**: Full TypeScript support with type inference for workflow inputs and outputs
- 🔄 **Workflow Orchestration**: Define complex workflows with dependencies and parallel execution
- 🔁 **Automatic Retries**: Configure retry policies for handling transient failures
- 📊 **Observability**: Track workflow progress and monitor execution metrics
- ⏰ **Scheduling**: Schedule workflows to run at specific times or on a recurring basis
- 🔄 **Event-Driven**: Trigger workflows based on events in your system

## Documentation

For detailed documentation, examples, and best practices, visit:
- [Hatchet Documentation](https://docs.hatchet.run)
- [Examples](https://github.com/hatchet-dev/hatchet/tree/main/sdks/typescript/src/v1/examples)

## Development

We use `pnpm` as our package manager. To get started with development:

1. Install dependencies:
```bash
pnpm install
```

2. Build the SDK:
```bash
pnpm build
```

3. Run tests:
```bash
pnpm test
```

## Contributing

We welcome contributions! Please check out our [contributing guidelines](https://docs.hatchet.run/contributing) and join our [Discord community](https://hatchet.run/discord) for discussions and support.

## License

This SDK is released under the MIT License. See [LICENSE](https://github.com/hatchet-dev/hatchet/blob/main/LICENSE) for details.
