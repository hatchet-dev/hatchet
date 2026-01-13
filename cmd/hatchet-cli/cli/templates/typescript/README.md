# Hatchet First Workflow Example

This is an example project demonstrating how to use Hatchet with TypeScript. For detailed setup instructions, see the [Hatchet Setup Guide](https://docs.hatchet.run/home/setup).

## Prerequisites

Before running this project, make sure you have the following:

1. [Node.js v16 or higher](https://nodejs.org/en/download)
2. npm, yarn, or pnpm package manager

## Setup

1. Clone the repository:

```bash
git clone https://github.com/hatchet-dev/hatchet-typescript-quickstart.git
cd hatchet-typescript-quickstart
```

2. Set the required environment variable `HATCHET_CLIENT_TOKEN` created in the [Getting Started Guide](https://docs.hatchet.run/home/hatchet-cloud-quickstart).

```bash
export HATCHET_CLIENT_TOKEN=<token>
```

> Note: If you're self hosting you may need to set `HATCHET_CLIENT_TLS_STRATEGY=none` to disable TLS

3. Install the project dependencies:

```bash
npm install
# or
yarn install
# or
pnpm install
```

### Running an example

1. Start a Hatchet worker:

```bash
npm run start
```

2. In a new terminal, run the example task:

```bash
npm run run:simple
```

This will trigger the task on the worker running in the first terminal and print the output to the second terminal.
