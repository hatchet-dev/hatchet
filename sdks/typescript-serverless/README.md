# @hatchet-dev/serverless

Serve Hatchet tasks from a serverless runtime without a worker process. Declare tasks with the
same API as the Hatchet TypeScript SDK, hand them to a runtime adapter, and register the
endpoint with Hatchet. The Hatchet serverless operator polls the endpoint for the workflows it
serves and delivers runs to it over signed HTTPS requests.

Status: preview, not yet published. This version serves non-durable and durable tasks on
Cloudflare Workers. The Vercel adapter, the `hatchet serverless` CLI commands and the
management module are the next phases.

## There is no Hatchet client in the serverless runtime

The package never reads a Hatchet API token and never instantiates a Hatchet client. Most
serverless runtimes cannot speak gRPC, and the transport that will replace it for them (a
ConnectRPC-style migration) is not decided. Everything a task needs arrives from the operator in
the signed request: input, parent outputs, retry count, metadata. Anything that would need a
client is unavailable and throws a `ServerlessLimitationError` naming the feature, which the
handler reports as a permanent failure (no retry, since retrying cannot help).

`ctx` members that throw `ServerlessLimitationError`:

| Member                                                                   | Reason                                                                                                           |
| ------------------------------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------- |
| `ctx.runChild`, `ctx.runNoWaitChild`, `ctx.spawnWorkflow`                | child runs need a client; durable tasks use `ctx.spawnChild` over the relay instead                              |
| `ctx.bulkRunChildren`, `ctx.bulkRunNoWaitChildren`, `ctx.spawnWorkflows` | same; durable tasks use `ctx.spawnChildren`                                                                      |
| `ctx.putStream`                                                          | streaming needs a client                                                                                         |
| `ctx.cancel`                                                             | cancellation is the operator's: it drops the request or closes the socket and the task's `abortController` fires |
| `ctx.refreshTimeout`, `ctx.releaseSlot`                                  | there is no worker slot to release or timeout to extend                                                          |
| `ctx.worker.labels`, `ctx.worker.upsertLabels`                           | there is no worker                                                                                               |

`ctx` members that work unchanged: `input`, `parentOutput`, `retryCount`, `workflowRunId`,
`taskRunExternalId`, `workflowNameV1`, `taskName`, `additionalMetadata`, `triggers`, `errors`,
`priority`, `triggeringEventId`, `triggeringEventKey`, `abortController` and `cancelled`.
`ctx.logger.*` and `ctx.log` print to the console only; no log line reaches the engine.
`ctx.worker.id()` is `undefined` and `ctx.worker.hasWorkflow(name)` answers from the
declarations you handed to the adapter. In a durable task, `ctx.now`, `ctx.sleepFor`,
`ctx.sleepUntil`, `ctx.waitFor`, `ctx.waitForEvent`, `ctx.spawnChild` and `ctx.spawnChildren`
work: the operator relays them to the engine over the invocation's websocket.

Declarations cannot run themselves either: `echo.run(...)`, `echo.schedule(...)` and
`echo.cron(...)` throw. Trigger runs from a process that has the regular SDK
(`HatchetClient.init().run(echo, input)` accepts a client-less declaration by name).

[LIMITATIONS.md](./LIMITATIONS.md) has the full list, including the declaration options the
handler ignores.

## Install

```sh
pnpm add @hatchet-dev/serverless @hatchet-dev/typescript-sdk zod
```

`@hatchet-dev/typescript-sdk` (1.32 or later) and `zod` are peer dependencies: the SDK supplies
the declaration API, `Context` and the registration format, nothing else.

## Declare tasks

Declarations are runtime-neutral: the same file compiles for a worker, a Cloudflare Worker or a
Vercel Function.

```ts
// src/tasks.ts
import { hatchet } from '@hatchet-dev/serverless';

export const echo = hatchet.task({
  name: 'echo',
  retries: 3,
  fn: async (input: { message: string }, ctx) => ({
    echo: input.message,
    runId: ctx.workflowRunId(),
    retry: ctx.retryCount(),
  }),
});

export const sleepThenEcho = hatchet.durableTask({
  name: 'sleep-then-echo',
  executionTimeout: '5m',
  fn: async (input: { message: string }, ctx) => {
    const startedAt = await ctx.now(); // memoized: replays after an eviction
    await ctx.sleepFor('3s'); // longer than the inline budget: evicts, re-invoked later
    const child = await ctx.spawnChild(echo, { message: 'from durable' }); // over the relay
    return { echo: input.message, startedAt, child, finishedAt: new Date().toISOString() };
  },
});

export const pipeline = hatchet.workflow<{ url: string }>({ name: 'pipeline' });
const fetchStep = pipeline.task({
  name: 'fetch',
  fn: async (input) => ({ body: await (await fetch(input.url)).text() }),
});
pipeline.task({
  name: 'summarize',
  parents: [fetchStep],
  fn: async (_, ctx) => ({ words: (await ctx.parentOutput(fetchStep)).body.split(' ').length }),
});

export const workflows = [echo, sleepThenEcho, pipeline];
```

`hatchet` is the SDK's `task`, `durableTask`, `workflow` and `batchTask` factories bound to no
client. A durable task runs on a websocket the operator dials; it may wait inline for the
endpoint's `inlineWaitBudgetMs` (5000 by default) and evicts itself past that, to be re-invoked
when the awaited sleep, event or child run completes. See LIMITATIONS.md for the rules.

## Cloudflare Workers

```ts
// src/index.ts
import { cloudflare } from '@hatchet-dev/serverless/cloudflare';
import { workflows } from './tasks';

export default cloudflare({ workflows });
```

```toml
# wrangler.toml
name = "orders"
main = "src/index.ts"
compatibility_date = "2026-09-01"
```

```sh
wrangler secret put HATCHET_SIGNING_SECRET   # 32+ characters; keep the value for the registration
wrangler deploy
```

`cloudflare({ workflows })` returns an `ExportedHandler<Env>` serving `POST /hatchet/healthcheck`,
`POST /hatchet/trigger` and the durable websocket upgrade on `/hatchet/trigger`. It reads
`HATCHET_SIGNING_SECRET` and the optional `HATCHET_ENDPOINT_ID` from `env`, and keeps the isolate
alive with `ctx.waitUntil` while a durable invocation is on its socket. Every other path gets a
404 unless you pass a `fetch` option.

```ts
export default cloudflare({
  workflows,
  serve: [echo], // actions this endpoint serves (default: all)
  basePath: '/internal/hatchet', // default "/hatchet"
  secret: (env) => env.ORDERS_SIGNING_SECRET, // default env.HATCHET_SIGNING_SECRET
  fetch: (request, env, ctx) => app.fetch(request, env, ctx), // everything not under basePath
});
```

The package's runtime entries (`.` and `./cloudflare`) carry no Node built-ins and bundle with
`wrangler deploy` without `nodejs_compat`.

## Register the endpoint

The `hatchet serverless register` command is the intended way and is not built yet. Until it
lands, call the REST API directly. The response carries the endpoint's `namespace`; everything
the endpoint registers is prefixed `<namespace>_`, so the `echo` task above is triggered as
`<namespace>_echo`.

```sh
curl -X POST https://<hatchet>/api/v1/stable/tenants/$TENANT_ID/serverless/endpoints \
  -H "Authorization: Bearer <api token>" -H 'Content-Type: application/json' \
  -d '{
    "name": "orders",
    "kind": "CLOUDFLARE_WORKERS",
    "healthcheckUrl": "https://orders.<account>.workers.dev/hatchet/healthcheck",
    "triggerUrl": "https://orders.<account>.workers.dev/hatchet/trigger",
    "signingSecret": "<the value given to wrangler secret put>"
  }'
```

The API token belongs to the process that registers the endpoint (a deploy script, your shell),
never to the Worker.

## Tests without Hatchet

`@hatchet-dev/serverless/testing` invokes a handler the way the operator would: it signs the
request, namespaces the action id and classifies the response with the operator's rules.

```ts
import { createTestOperator } from '@hatchet-dev/serverless/testing';
import { echo, workflows } from './tasks';

const op = createTestOperator({ workflows, secret: 'test-secret-at-least-32-characters-long' });

test('echo', async () => {
  expect(await op.invoke(echo, { message: 'hi' })).toMatchObject({ echo: 'hi' });
});

test('healthcheck is protojson', async () => {
  const body = await op.healthcheck();
  expect(body.workflows.map((w) => w.name)).toEqual(['echo']);
});
```

`op.deliver(...)` returns the operator's classification (`completed` with the output, or `failed`
with the message and retry decision) instead of throwing; `op.request(path, init)` sends a raw
signed request to the handler.

Durable tasks run against an in-memory operator with a virtual clock and an event log:

```ts
test('sleep-then-echo evicts and resumes', async () => {
  const first = await op.invokeDurable(
    sleepThenEcho,
    { message: 'hi' },
    { inlineWaitBudgetMs: 100 }
  );
  expect(first.status).toBe('evicted');
  expect(first.endpointFrames).toEqual([
    'memo',
    'completeMemo',
    'waitFor',
    'evictInvocation',
    'done:evicted',
  ]);

  op.clock.advance('3s');

  const second = await op.resume(first);
  expect(second.status).toBe('completed');
  expect(second.output).toMatchObject({ echo: 'hi' });
});
```

`op.startDurable(...)` returns the invocation in progress, with `frame(kind)` to wait for a
frame the endpoint sent, and `serverEvict()`, `sendError()` and `close()` to act as the operator
mid-flight; `op.emit(eventKey, payload)` satisfies a `waitForEvent`. The fake enforces the
operator's protocol rules (one ack-bearing request in flight, nothing after the done frame,
`done: evicted` only after an eviction ack), so a violation fails the test.

## What the package verifies

Every request from the operator is signed with the endpoint's secret (`X-Hatchet-Signature`,
hex HMAC-SHA256). The handler checks, in this order:

- **POST bodies** (healthcheck, non-durable trigger): the HMAC over the raw body, then the
  `timestamp` field the signature covers, which must be within five minutes of the handler's
  clock in either direction (`REQUEST_MAX_AGE_SECONDS`), then the `endpointId` when the
  handler is configured with one (`HATCHET_ENDPOINT_ID` on Cloudflare). A captured request is
  therefore only replayable for five minutes; a task with side effects should treat
  `(endpointId, taskRunExternalId, retryCount)` as its idempotency key.
- **Websocket upgrades** (durable trigger): the `X-Hatchet-Endpoint-Id` header must be present
  and, when the handler is configured with an id, equal to it; the same five minute window
  applies to `X-Hatchet-Timestamp`; the HMAC covers `endpointId.timestamp.nonce.taskId.invocation`
  with the header's endpoint id, so a signature made for one endpoint cannot be presented to
  another that shares the secret; then the nonce is consumed from a bounded in-memory set only
  after the signature verified (`NonceSet`, 4096 entries). Every accepted nonce is kept until
  its window expires and nothing is dropped early to make room: when the set is full of
  unexpired nonces the upgrade is answered `503 {"error": "nonce store full", "retry": true}`
  and the operator retries later. The set is per isolate; a production endpoint should consume
  nonces in a Durable Object or an expiring KV key through the `seenNonce` option (return
  `true` for a replay, `'full'` when there is no room), so a replay that lands in another
  isolate is caught too.
- **The first frame**: its task id and invocation must match the verified upgrade headers,
  otherwise the socket is closed with 1008 before any task code runs.

## How a trigger maps onto a response

The operator (`pkg/serverlessoperator/delivery.go`) reads the status code; a
`{"error", "retry"}` body overrides its message and retry decision.

| Task outcome                                            | Response                           |
| ------------------------------------------------------- | ---------------------------------- |
| returns a value                                         | `200`, the value as JSON           |
| returns `undefined`                                     | `204`                              |
| throws `NonRetryableError`                              | `422 {"error", "retry": false}`    |
| throws `ServerlessLimitationError`                      | `422 {"error", "retry": false}`    |
| throws anything else                                    | `500 {"error", "retry": true}`     |
| action not served here                                  | `404 {"error", "retry": false}`    |
| durable action sent as a POST                           | `422 {"error", "retry": false}`    |
| bad signature, or a timestamp outside the window        | `401 {"error", "retry": false}`    |
| endpoint id mismatch                                    | `403 {"error", "retry": false}`    |
| durable upgrade with a bad signature or stale timestamp | `401`; a foreign endpoint id `403` |
| durable upgrade on an adapter without the relay         | `426 {"error", "retry": false}`    |

On the durable websocket the outcome travels in the final `done` frame: `{ output }` completes
the task, `{ error, retry }` fails it (retry is false for `NonRetryableError` and after an engine
error frame), and `{ status: "evicted" }` after an eviction ack means the engine re-invokes the
task later.

## Development

The package depends on the SDK's build output (`file:../typescript/dist`) while 1.32 is
unreleased, so build the SDK first and reinstall after every SDK rebuild:

```sh
pnpm run build:sdk        # builds ../typescript/dist, which pnpm copies into node_modules
pnpm install
pnpm lint && pnpm typecheck && pnpm test
pnpm check:edge           # builds with tsup, then proves the runtime entries reach no Node built-in
```

### Generated code

`src/generated/proto/` is ts-proto output for `api-contracts/v1/serverless.proto` and the
messages it imports (`v1/dispatcher.proto`, `v1/workflows.proto`, `v1/shared/*.proto` and the
package-less `dispatcher.proto`). The operator encodes the same messages with protojson, so
`fromJSON` and `toJSON` on the generated types read and write exactly what the operator sends and
expects. Regenerate after changing any of those files:

```sh
pnpm run generate-proto
```

CI regenerates this directory (`task generate-serverless-proto`, part of `task generate-all`)
and fails when the committed output is stale.
