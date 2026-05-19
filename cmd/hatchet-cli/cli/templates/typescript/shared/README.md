# Hatchet TypeScript Quickstart - {{ .Name }}

This is an example project demonstrating how to use Hatchet with TypeScript. For detailed setup instructions, see the [Hatchet Setup Guide](https://docs.hatchet.run/home/setup).

## Prerequisites

Before running this project, make sure you have the following:

1. [Node.js v16 or higher](https://nodejs.org/en/download)
{{- if eq .PackageManager "npm"}}
2. npm package manager (included with Node.js)
{{- else if eq .PackageManager "pnpm"}}
2. [pnpm](https://pnpm.io/installation) package manager
{{- else if eq .PackageManager "yarn"}}
2. [Yarn](https://yarnpkg.com/getting-started/install) package manager
{{- else if eq .PackageManager "bun"}}
2. [Bun](https://bun.sh/) runtime and package manager
{{- end}}

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
{{- if eq .PackageManager "npm"}}
npm install
{{- else if eq .PackageManager "pnpm"}}
pnpm install
{{- else if eq .PackageManager "yarn"}}
yarn install
{{- else if eq .PackageManager "bun"}}
bun install
{{- end}}
```

### Running an example

1. Start a Hatchet worker:

```bash
{{- if eq .PackageManager "npm"}}
npm run start
{{- else if eq .PackageManager "pnpm"}}
pnpm start
{{- else if eq .PackageManager "yarn"}}
yarn start
{{- else if eq .PackageManager "bun"}}
bun start
{{- end}}
```

2. In a new terminal, run the example task:

```bash
{{- if eq .PackageManager "npm"}}
npm run run:simple
{{- else if eq .PackageManager "pnpm"}}
pnpm run:simple
{{- else if eq .PackageManager "yarn"}}
yarn run:simple
{{- else if eq .PackageManager "bun"}}
bun run:simple
{{- end}}
```

This will trigger the task on the worker running in the first terminal and print the output to the second terminal.
