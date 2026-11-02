# @hatchet-dev/sandbox

<div align="center">

[![npm version](https://img.shields.io/npm/v/%40hatchet-dev%2Fsandbox)](https://www.npmjs.com/package/@hatchet-dev/sandbox)
[![License: MIT](https://img.shields.io/badge/License-MIT-purple.svg)](https://opensource.org/licenses/MIT)

</div>

Run agent sandboxes from [Hatchet](https://hatchet.run) tasks.

A sandbox is a workspace with a shell. You run commands in it and read and write its files. Each call is a Hatchet task. Each call that changes files takes a snapshot. You can start a new sandbox from any snapshot, and you can come back to a sandbox later by its ID. If you use it inside a durable task, a crash picks up where it left off.

## Install

```bash
pnpm add @hatchet-dev/sandbox @hatchet-dev/typescript-sdk
```

```bash
npm install @hatchet-dev/sandbox @hatchet-dev/typescript-sdk
```

```bash
yarn add @hatchet-dev/sandbox @hatchet-dev/typescript-sdk
```

You need `@hatchet-dev/typescript-sdk` 1.33.1 or newer.

## Quick start

Import the package once. It adds `hatchet.sandbox` to the Hatchet client.

```ts
import { HatchetClient } from '@hatchet-dev/typescript-sdk/v1';
import '@hatchet-dev/sandbox';

const hatchet = HatchetClient.init();

const session = hatchet.durableTask({
  name: 'agent-session',
  executionTimeout: '1h',
  fn: async (input: { conversationId: string }) => {
    const sandbox = await hatchet.sandbox.create({ id: input.conversationId });
    await sandbox.files.write('haiku.txt', 'an old silent pond');
    const haiku = await sandbox.exec('cat haiku.txt');
    await sandbox.kill();
    return { haiku: haiku.stdout };
  },
});

// later, from anywhere with a Hatchet client
const again = await hatchet.sandbox.connect('conv-42');
await again.exec('cat haiku.txt');
```

Your app only talks to Hatchet. A brickie provider, connected to the same tenant, runs the sandboxes. Set `HATCHET_CLIENT_TOKEN` like you would for any Hatchet client. That is all the app needs.

## API

### `hatchet.sandbox`

| Call                                | Returns         | What it does                                                                                                       |
| ----------------------------------- | --------------- | ------------------------------------------------------------------------------------------------------------------ |
| `create(opts?)`                     | `Sandbox`       | Starts a sandbox.                                                                                                  |
| `connect(id, { template? })`        | `Sandbox`       | Picks up a sandbox from its latest snapshot. Starts it again if it was stopped. Throws if the ID does not exist.   |
| `list({ sandboxId?, metadata? })`   | `SandboxInfo[]` | Finds sandboxes by ID or by metadata. All given pairs must match.                                                  |
| `delete({ sandboxId?, metadata? })` | `string[]`      | Deletes every matching sandbox with all of its snapshots and files, and returns their IDs. One filter is required. |

Options for `create`:

| Option                  | What it does                                                                                                                 |
| ----------------------- | ---------------------------------------------------------------------------------------------------------------------------- |
| `id`                    | Pick your own sandbox ID. Lowercase letters, digits, and hyphens. It must not exist yet.                                     |
| `template`              | The container image. Defaults to the provider's image.                                                                       |
| `from`                  | Start from this snapshot. Create two sandboxes from the same snapshot and you have a fork.                                   |
| `metadata`              | Tags you can search for with `list`.                                                                                         |
| `inheritParentMetadata` | With `from`, copy the tags of the sandbox that made the snapshot onto the new one. Tags given in `metadata` take precedence. |
| `idleTimeoutMs`         | Remove the sandbox after this long with no calls. The provider sets the maximum.                                             |

### `Sandbox`

| Call                        | Returns                                               | What it does                                                                                                                                    |
| --------------------------- | ----------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------- |
| `exec(cmd, { timeoutMs? })` | `{ stdout, stderr, exitCode, snapshotId?, timings? }` | Runs `sh -c cmd` in the workspace and takes a snapshot. A non-zero exit code is a normal result, not an error. On timeout the exit code is 124. |
| `kill()`                    | `void`                                                | Stops the sandbox. The snapshots stay, so `connect` still works.                                                                                |
| `delete()`                  | `void`                                                | Stops the sandbox and deletes every snapshot and file stored for it. `connect` fails afterwards.                                                |
| `sandboxId`                 | `string`                                              | The ID to use with `connect` later.                                                                                                             |
| `files`                     | `Files`                                               | The file API below.                                                                                                                             |

Each result also has `timings` in milliseconds for `claim`, `restore`, `exec`, and `checkpoint`, and the `snapshotId` the call made.

### `sandbox.files`

Paths are relative to the workspace. A path that points outside it throws a `FilesystemError` with code `invalid_path`.

| Call                                                                  | What it does                                                                                      |
| --------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------- |
| `read(path, { format? })`                                             | Reads a file. `format` is `'text'` (default), `'bytes'` for a `Uint8Array`, or `'blob'`.          |
| `write(path, data)` or `write([{ path, data }, ...])`                 | Writes one file or many. `data` can be a string, `ArrayBuffer`, `Uint8Array`, or `Blob`.          |
| `list(path?, { depth? })`                                             | Lists entries with name, path, type, size, mode, permissions, and modified time.                  |
| `exists(path)`, `getInfo(path)`                                       | Checks a path or gets its info.                                                                   |
| `makeDir(path)`, `rename(from, to)`, `copy(from, to)`, `remove(path)` | The usual file operations.                                                                        |
| `download(url, path, { timeoutMs? })`                                 | The provider fetches an https URL into the workspace. Private and loopback addresses are refused. |
| `upload(path, url, { timeoutMs? })`                                   | The provider sends a file to an https URL with HTTP PUT. Presigned URLs work well here.           |

The sandbox itself does not need network access for any of this.

`read` and `write` carry up to 2 MiB through Hatchet. Bigger files go through the provider's object store (`--object-store` on the provider: S3, GCS, Azure Blob, or a local folder) and never pass through Hatchet. You call the same methods either way.

## Errors

- `FilesystemError` has a `code`. One of: `not_found`, `invalid_path`, `invalid_url`, `invalid_content`, `invalid`, `is_a_directory`, `not_a_directory`, `too_large`, `staging_failed`, `transfer_failed`.
- A command that fails does not throw. Look at `exitCode`.
- `connect` with an unknown ID throws. `create` with an `id` that already exists throws too. Use `connect` for that one, or pick a new ID.
- `hatchet.sandbox.delete` without a `sandboxId` or a non-empty `metadata` filter throws before touching anything. Metadata matching is a subset match, the same as `list`: a filter of `{ a, b }` matches a sandbox tagged `{ a, b, c }`.

## Telemetry

On first use the SDK turns on the Hatchet OpenTelemetry instrumentor. If you have OpenTelemetry set up, sandbox calls show up as spans in your task's trace. Set `BRICKIE_OTEL=false` to turn this off.

## What you need

- A Hatchet tenant. Hatchet Cloud, or a self-hosted Hatchet v0.110.0 or newer with gRPC operators turned on.
- A brickie provider connected to that tenant. It is one binary, `brickie worker`. Get it from [github.com/hatchet-dev/brickie](https://github.com/hatchet-dev/brickie).

## Changelog

See [CHANGELOG.md](https://github.com/hatchet-dev/hatchet/blob/main/sdks/typescript-sandbox/CHANGELOG.md).

## License

MIT. See [LICENSE](https://github.com/hatchet-dev/hatchet/blob/main/sdks/typescript-sandbox/LICENSE).
