# @hatchet-dev/pi-durable

Run [pi-durable](https://www.npmjs.com/package/@earendil-works/pi-durable) agents on Hatchet.

pi-durable keeps an agent's state durable: conversations, tool calls in progress, forks, and documents all survive a
crash. This package gives it the rest of what a production agent needs:

- **A home for each session.** `defineSessionTask` makes a Hatchet task that owns one session at a time. Concurrency on
  the session id means there is never more than one owner. If a worker dies, Hatchet retries the run on another worker,
  and pi-durable resumes the session from its storage.
- **A sandbox for each conversation.** `sandboxCoding()` gives the agent pi's read, write, edit and bash tools, and
  `sandboxEnvs()` runs every tool call in a [Hatchet sandbox](https://www.npmjs.com/package/@hatchet-dev/sandbox) instead
  of on the worker. A fork gets its own sandbox, restored from the snapshot taken at the fork point.
- **A way in and out.** `SessionClient` talks to a session from anywhere (a web server, a CLI, another task) over two
  Hatchet streams: an inbox it publishes to and a view it reads.

> pi-durable is experimental. This package pins it to `~1.5.0`, because a minor release can change its storage format
> or API.

## Install

```bash
pnpm add @hatchet-dev/pi-durable @hatchet-dev/typescript-sdk @hatchet-dev/sandbox \
  @earendil-works/pi-durable@~1.5.0 @earendil-works/pi-ai@~1.5.0 @earendil-works/chord@~1.5.0
```

Your tenant needs a sandbox provider, such as brickie. The engine must have operators enabled
(`SERVER_GRPC_OPERATORS_ENABLED`, Hatchet v0.110.0 or newer).

## Use

```ts
import { createModels } from '@earendil-works/pi-ai/models';
import { anthropicProvider } from '@earendil-works/pi-ai/providers/anthropic';
import { createRegistry, Harness } from '@earendil-works/pi-durable';
import { openNodeSqliteStorage } from '@earendil-works/pi-durable/storage/sqlite/node';
import { HatchetClient } from '@hatchet-dev/typescript-sdk/v1';
import {
  defineSessionTask,
  SessionClient,
  sandboxCoding,
  sandboxEnvs,
} from '@hatchet-dev/pi-durable';

const hatchet = HatchetClient.init();
const models = createModels();
models.setProvider(anthropicProvider());
const registry = createRegistry();
registry.install(sandboxCoding({ instructions: 'The repository is in /workspace.' }));

// The worker: one run of this task owns one session.
export const session = defineSessionTask(hatchet, {
  agent: { model: { provider: 'anthropic', modelId: 'claude-opus-5-5' }, cwd: '/workspace' },
  sandbox: { template: 'node:24-alpine' },
  openHarness: async (sessionId, onReport, context) =>
    Harness.open(
      await openNodeSqliteStorage(`.pi-data/${sessionId}.sqlite`),
      { models, registry, env: sandboxEnvs(hatchet, { template: 'node:24-alpine' }), onReport },
      context
    ),
});
await (await hatchet.worker('pi-worker', { workflows: [session] })).start();

// Anywhere else:
const sessions = new SessionClient(hatchet);
const id = await sessions.create();
await sessions.send(id, { type: 'input', conversationId: 1, text: 'Fix the failing test.' });
for await (const view of sessions.views(id)) console.log(view.conversations[0]?.items.at(-1));
```

### Messages

`send` takes one of these. A session starts with one conversation, the root, whose id is 1; forks add more.

| Message                                              | Effect                                                                                                                                                                                               |
| ---------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `{ type: 'input', conversationId, text, whenBusy? }` | Adds a user message. A busy conversation gets it as a follow-up after the current run (the default) or as a steer during it (`whenBusy: 'steer'`). A conversation's first input creates its sandbox. |
| `{ type: 'fork', conversationId, entryId }`          | Starts a new conversation from that entry, with a sandbox restored from the snapshot of that point.                                                                                                  |
| `{ type: 'abort', conversationId }`                  | Stops the conversation's current run.                                                                                                                                                                |

Each message carries a request id, so a message that is delivered twice is applied once. A message the host cannot
apply, such as an input for a conversation that does not exist, is not dropped: the view's `rejected` list has it, with
the reason.

### Views

The host publishes a `SessionView` after every change: the host's run and attempt, for each conversation its items,
its live generation and tool output, its sandbox and snapshot, and its token usage and cost, and the last 20 messages
it rejected. A reader that connects starts at the newest view.

### The sandbox environment

`SandboxEnv` passes pi-durable's `ExecutionEnv` conformance suite (24 cases) against a real sandbox. Where a sandbox
works differently from a local process:

- A command's output arrives in one chunk per stream when the command ends, not as it runs.
- Aborting a command cancels its sandbox run, and the provider kills the command. Cancelling uses Hatchet's REST API.
- `watch` polls: one sandbox command per `watchIntervalMs` (default 500 ms) compares every watched path. Each poll also
  makes a snapshot, so watch only what you need. Two writes of the same size within one second can look alike.
- A sandbox runs one call at a time.
- `appendFile`, `truncateFile`, `flushFile`, `openTextLineReader`, `createTempDir` and `createTempFile` are not
  supported. pi's coding tools do not use them.

### Storage

The host does not store anything itself: failover reaches as far as the storage you open in `openHarness`. SQLite or
JSONL on the worker's disk only fails over to a worker on the same machine (or one with the same volume). pi-durable
has no cross-process locking, so there must be one owner at a time. The task's concurrency gives you that.

### Options

`defineSessionTask(hatchet, options)`:

| Option                  | Default      |                                                                                                           |
| ----------------------- | ------------ | --------------------------------------------------------------------------------------------------------- |
| `name`                  | `pi-session` | The task's name, and the prefix of the session's stream topics. Give `SessionClient` the same `taskName`. |
| `openHarness`           | required     | Opens the session's harness.                                                                              |
| `agent`                 | required     | The root conversation's model, thinking level and working directory.                                      |
| `sandbox.template`      | required     | The image every conversation's sandbox runs.                                                              |
| `sandbox.workspace`     | `/workspace` | The sandbox's workspace. Tool paths outside it are refused.                                               |
| `sandbox.idleTimeoutMs` | 30 min       | A sandbox with no calls for this long stops. The next call restarts it from its snapshot.                 |
| `sandbox.seed`          | none         | Fills a conversation's first sandbox, for example with a repository.                                      |
| `maxSessionCostUsd`     | no cap       | Aborts every conversation once the session has spent this much.                                           |
| `idleGraceMs`           | 20 s         | How long an idle host waits for the next message before it hands the session back.                        |
| `stallMs`               | 15 s         | A ready task that has not started after this long means the owner is stuck.                               |
| `retries`               | 3            | Retries of the task after a crash.                                                                        |
| `executionTimeout`      | `60m`        | The longest one run may own the session.                                                                  |

## Example

`examples/coding-agent` is a coding agent that fixes a failing test in a small repository. It has a web page that
shows the conversation, the live tool output, and the host's attempt, so you can kill the worker and watch the session
come back.

The example runs from this directory, against the SDK built from this repository (see [Develop](#develop)).

```bash
cp .env.example .env   # fill in HATCHET_CLIENT_TOKEN and ANTHROPIC_API_KEY
pnpm example:worker    # in one terminal; start a second one to see failover
pnpm example:server    # in another, then open http://localhost:3334
```

For a local engine whose API certificate your system trusts but Node does not (a local mkcert certificate, say), set
`NODE_OPTIONS=--use-system-ca` in `.env`. Without it, cancelling an aborted command and scheduling a wakeup fail.

## Develop

The package links `@hatchet-dev/typescript-sdk` and `@hatchet-dev/sandbox` from their build output in `../typescript`
and `../typescript-sandbox`, so build both before installing:

```bash
pnpm build:sdk                  # builds ../typescript into ../typescript/dist
pnpm build:sandbox              # builds ../typescript-sandbox into ../typescript-sandbox/dist
pnpm install
pnpm lint
pnpm typecheck
pnpm test                       # unit tests
PI_SANDBOX_TESTS=1 pnpm test    # also pi-durable's ExecutionEnv conformance suite, against a real sandbox
pnpm build
```

## Release

Bump `version` in `package.json` and move the `[Unreleased]` notes in `CHANGELOG.md` under the new version. On a push
to `main`, CI publishes the version to npm if it is not there yet and creates a `pi-durable/<version>` GitHub release.
A version containing `alpha` is published under the `alpha` tag.

## License

MIT
