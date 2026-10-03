# @hatchet-dev/serverless

Serve Hatchet tasks from a serverless runtime without a worker process. Declare tasks with the
same API as the Hatchet TypeScript SDK, hand them to a runtime adapter, and register the
endpoint with Hatchet. The Hatchet serverless operator polls the endpoint for the workflows it
serves and delivers runs to it over signed HTTPS requests.

Status: preview, not yet published. This version serves non-durable and durable tasks on
Cloudflare Workers, Vercel (Next.js App Router or a standalone function) and a Node server you
run yourself, with an optional Hatchet client for child runs, streams and logs. The
`hatchet serverless` CLI commands and the management module are the next
phases.

## The client, and what `ctx` can do

Everything a task needs to run arrives from the operator in the signed request: input, parent
outputs, retry count, metadata. Everything a task does that reaches the engine goes through
the handler's optional `client`: the SDK's core client (`HatchetCore` from
`@hatchet-dev/typescript-sdk/core`), which makes unary Connect calls over `fetch` and so runs
wherever the handler runs. You configure it with a tenant API token from the platform's secret
store, next to the signing secret; the operator never sends a token to the endpoint.

```ts
export default cloudflare({ workflows, streams: [parent] }); // client from env.HATCHET_CLIENT_TOKEN
createHandler({ workflows, client: { token: env.HATCHET_CLIENT_TOKEN }, ... });
createHandler({ workflows, client: (env) => ({ token: env.HATCHET_CLIENT_TOKEN }), ... });
createHandler({ workflows, client: new HatchetCore({ token, serverUrl, tls }), ... });
```

`client` takes a `HatchetCore`, the config it takes (`{ token, serverUrl?, hostPort?, tls?,
namespace?, ... }`; with only a `token` the engine address comes from the token), or a function
of the runtime's environment returning either, for runtimes that hand secrets to the request.
One client is built per handler (per environment object for the function form) the first time
a task needs it, and reused.

With a client:

| Member                                                                  | How it works                                                                                                                                                |
| ----------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `ctx.runNoWaitChild`, `ctx.bulkRunNoWaitChildren`, `ctx.spawnWorkflow*` | one unary `TriggerWorkflow` per child through the client, with the parent run, index and key set                                                            |
| `ctx.runChild`, `ctx.bulkRunChildren`, `ref.result()` inside a task     | the trigger as above; the result is awaited on the invocation websocket, where the operator relays one `SubscribeToWorkflowRuns` engine stream for the task |
| `ctx.putStream`, `ctx.cancel`                                           | `PutStreamEvent` and `CancelTasks` through the client                                                                                                       |
| `ctx.log`                                                               | the console, and the line is written to the engine too                                                                                                      |
| `ctx.refreshTimeout`, `ctx.releaseSlot`, `ctx.worker.labels`            | still throw: there is no worker                                                                                                                             |

Awaiting a child needs the invocation websocket, which the operator opens only for the tasks
the catalog asks one for: every durable task, and every non-durable task you list in the
`streams` option (`streams: [parent]`, or action ids like `'pipeline:summarize'`). The
healthcheck advertises them as `tasks: [{ action, streams: true }]`; every other task is
delivered over a signed POST, which is cheaper and has nothing to await on, so `result()` there
throws and names the option. The operator allows at most 16 engine streams per socket and the
package keeps every child of a task on one of them. Inside a durable task `ctx.runChild` throws:
use `ctx.spawnChild` and `ctx.spawnChildren`, which record the child in the durable event log
and replay its result after an eviction.

Without a client every one of these members throws a `ServerlessLimitationError` naming the
feature and "configure `client` to enable this", which the handler reports as a permanent
failure (no retry, since retrying cannot help). `ctx.logger.*` and `ctx.log` print to the
console. Existing deployments without a client are unchanged.

`ctx` members that never need a client: `input`, `parentOutput`, `retryCount`, `workflowRunId`,
`taskRunExternalId`, `workflowNameV1`, `taskName`, `additionalMetadata`, `triggers`, `errors`,
`priority`, `triggeringEventId`, `triggeringEventKey`, `abortController` and `cancelled`.
`ctx.worker.id()` is `undefined` and `ctx.worker.hasWorkflow(name)` answers from the
declarations you handed to the adapter. In a durable task, `ctx.now`, `ctx.sleepFor`,
`ctx.sleepUntil`, `ctx.waitFor`, `ctx.waitForEvent`, `ctx.spawnChild` and `ctx.spawnChildren`
work with or without a client: the operator relays them to the engine over the invocation's
websocket.

Declarations cannot run themselves: `echo.run(...)`, `echo.schedule(...)` and `echo.cron(...)`
throw. Trigger runs through a `HatchetCore` (`new HatchetCore({ token }).run(echo, input)`
accepts the declaration; `HatchetCore` is re-exported from `@hatchet-dev/serverless`) or from a
process that has the regular SDK.

[LIMITATIONS.md](./LIMITATIONS.md) has the full rules, including the declaration options the
handler ignores.

## Install

```sh
pnpm add @hatchet-dev/serverless @hatchet-dev/typescript-sdk zod
```

`@hatchet-dev/typescript-sdk` (1.34.0-alpha.2 or later, for its `/core` entry) and `zod` are
peer dependencies: the SDK supplies the declaration API, `Context`, the registration format and
the core client, nothing else.

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

// Needs the client and, since it awaits the child, a socket: list it in `streams`.
export const parent = hatchet.task({
  name: 'parent',
  fn: async (input: { message: string }, ctx) => ({
    child: await ctx.runChild(echo, { message: `${input.message} (from parent)` }),
  }),
});

export const workflows = [echo, sleepThenEcho, pipeline, parent];
```

`hatchet` is the SDK's `task`, `durableTask`, `workflow` and `batchTask` factories bound to no
client. A durable task runs on a websocket the operator dials; it may wait inline for the
endpoint's `inlineWaitBudgetMs` (5000 by default) and evicts itself past that, to be re-invoked
when the awaited sleep, event or child run completes. A non-durable task listed in `streams`
runs on a websocket too, waits for its children inline and never evicts. See LIMITATIONS.md for
the rules.

## Cloudflare Workers

```ts
// src/index.ts
import { cloudflare } from '@hatchet-dev/serverless/cloudflare';
import { workflows } from './tasks';

export default cloudflare({ workflows, streams: [parent] });
```

```toml
# wrangler.toml
name = "orders"
main = "src/index.ts"
compatibility_date = "2026-09-01"
```

```sh
wrangler secret put HATCHET_SIGNING_SECRET   # 32+ characters; keep the value for the registration
wrangler secret put HATCHET_CLIENT_TOKEN     # a tenant API token; optional, for the ctx methods above
wrangler deploy
```

`cloudflare({ workflows })` returns an `ExportedHandler<Env>` serving `POST /hatchet/healthcheck`,
`POST /hatchet/trigger` and the invocation websocket upgrade on `/hatchet/trigger`. It reads
`HATCHET_SIGNING_SECRET`, the optional `HATCHET_CLIENT_TOKEN` (the client's token; without it
there is no client) and the optional `HATCHET_ENDPOINT_ID` from `env`, and keeps the isolate
alive with `ctx.waitUntil` while an invocation is on its socket. Every other path gets a 404
unless you pass a `fetch` option.

```ts
export default cloudflare({
  workflows,
  serve: [echo, parent], // actions this endpoint serves (default: all)
  streams: [parent], // non-durable tasks invoked over the socket, so they can await children
  basePath: '/internal/hatchet', // default "/hatchet"
  secret: (env) => env.ORDERS_SIGNING_SECRET, // default env.HATCHET_SIGNING_SECRET
  client: (env) => ({ token: env.HATCHET_CLIENT_TOKEN }), // the default; add serverUrl and tls for a local engine
  fetch: (request, env, ctx) => app.fetch(request, env, ctx), // everything not under basePath
});
```

The package's runtime entries (`.` and `./cloudflare`) carry no Node built-ins and bundle with
`wrangler deploy` without `nodejs_compat`.

## Vercel and Next.js

```ts
// app/api/hatchet/[...hatchet]/route.ts
import { vercel } from '@hatchet-dev/serverless/vercel';
import { workflows } from '@/hatchet/tasks';

export const { GET, POST } = vercel({ workflows });
export const maxDuration = 300; // the ceiling on one durable invocation; 800 on Pro and Enterprise
```

```sh
pnpm add @vercel/functions ws          # only for durable tasks; optional peers of the package
vercel env add HATCHET_SIGNING_SECRET  # 32+ characters; keep the value for the registration
```

`vercel({ workflows })` returns the two route handlers of a Next.js App Router catch-all route.
`POST` serves the healthcheck and non-durable triggers; `GET` is the durable websocket upgrade,
accepted through `experimental_upgradeWebSocket` from `@vercel/functions`, which needs
[Fluid compute](https://vercel.com/docs/functions/websockets) and the `ws` package. The route
is read from the catch-all segment, so the route file may live anywhere under `app/`; register
the endpoint with the URLs the file produces (`/api/hatchet/healthcheck` and
`/api/hatchet/trigger` for the file above). The signing secret comes from
`process.env.HATCHET_SIGNING_SECRET`, the optional endpoint id from `HATCHET_ENDPOINT_ID`.

```ts
export const { GET, POST } = vercel({
  workflows,
  serve: [echo], // actions this endpoint serves (default: all)
  secret: () => process.env.ORDERS_SIGNING_SECRET, // default process.env.HATCHET_SIGNING_SECRET
  durable: false, // no relay: for a project without Fluid compute (default: on when VERCEL is set)
  basePath: '/hooks/hatchet', // only for calls without a route context (default "/api/hatchet")
});
```

What the adapter does with the socket: the upgrade callback awaits the relay, so the route's
response completes only when the invocation does, and the same promise goes to `waitUntil`
from `@vercel/functions` so the runtime accounts for it as background work. `after()` from
`next/server` is not used, because it schedules work for after the response while the relay is
already running when the 101 goes out, and because the adapter also serves functions outside
Next.js. The socket's `maxPayload` is the operator's 4 MiB frame limit.

When the upgrade cannot happen the adapter never hangs: without `@vercel/functions` or `ws`,
or with `durable: false`, or outside Vercel (no `VERCEL` environment variable, which is what
`next dev` looks like), the healthcheck reports `durable.supported: false` and the operator
sends only non-durable tasks. A project without Fluid compute cannot be told apart from a
healthcheck, so there an upgrade is refused at request time with
`426 {"error": "websocket upgrade unavailable: ...", "retry": false}` and a logged hint; pass
`durable: false` on such a project. Locally, the websocket path runs only under `vc dev`
(Vercel CLI 54.14.2 or later), not `next dev`.

Constraints Vercel imposes: a durable invocation cannot outlive `maxDuration` (300 s on Hobby,
800 s on Pro and Enterprise), so the endpoint's `inlineWaitBudgetMs` must stay well below it;
request and response bodies are capped at 4.5 MB.

A standalone function (`api/hatchet.ts`, any framework or none) exports the endpoint as an
`http.Server`, the shape Vercel serves directly, with the websocket through `ws`:

```ts
// api/hatchet.ts
import { vercelServer } from '@hatchet-dev/serverless/vercel';
import { workflows } from '../hatchet/tasks';

export default vercelServer({ workflows }); // routes under /hatchet
```

Vercel routes only `/api/hatchet` itself to that file, so `vercel.json` needs
`{ "rewrites": [{ "source": "/hatchet/:path*", "destination": "/api/hatchet" }] }` (or pass
`basePath: '/api/hatchet'` and rewrite `/api/hatchet/:path*`).

## Node

For a Node process you run yourself: a Next.js custom server, an Express app, a container.

```ts
import { createServer } from '@hatchet-dev/serverless/node';
import { workflows } from './tasks';

createServer({ workflows }).listen(3000); // POST /hatchet/healthcheck, /hatchet/trigger and the upgrade
```

`nodeHandler({ workflows })` is the same endpoint as an `http` request listener with the
websocket `upgrade` listener attached, for a server you already have. It calls `next()` outside
`basePath` when mounted as Express middleware (and honours the mount path), answers 404 there
otherwise, and `upgrade` returns false, leaving the socket untouched, for upgrades that are not
its own:

```ts
import express from 'express';
import { createServer } from 'node:http';
import { nodeHandler } from '@hatchet-dev/serverless/node';

const hatchet = nodeHandler({ workflows });
const app = express().use(hatchet);
const server = createServer(app);

server.on('upgrade', (req, socket, head) => {
  if (!hatchet.upgrade(req, socket, head)) socket.destroy();
});
server.listen(3000);
```

Both read `process.env.HATCHET_SIGNING_SECRET` and the optional `HATCHET_ENDPOINT_ID`, and take
the same `secret`, `endpointId`, `serve`, `basePath`, `durable` and `maxPayload` options as the
Vercel adapter. The relay needs the `ws` package (an optional peer); without it, or with
`durable: false`, the healthcheck reports `durable.supported: false`.

## Adapters

| Runtime                    | Import                                                            | Durable tasks served by                                                                    | Constraints                                                                                                                                                    |
| -------------------------- | ----------------------------------------------------------------- | ------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Cloudflare Workers         | `cloudflare` from `@hatchet-dev/serverless/cloudflare`            | `WebSocketPair`, kept alive with `ctx.waitUntil`; nothing to install                       | CPU time per invocation (`limits.cpu_ms`, 5 min on Workers Paid); nonces per isolate unless `seenNonce` is backed by a Durable Object or KV                    |
| Vercel, Next.js App Router | `vercel` from `@hatchet-dev/serverless/vercel`                    | `experimental_upgradeWebSocket` from `@vercel/functions` plus `ws`; Fluid compute required | `maxDuration` caps an invocation (300 s Hobby, 800 s Pro and Enterprise); 4.5 MB bodies; websocket locally only under `vc dev`; `durable: false` without Fluid |
| Vercel, standalone `api/`  | `vercelServer` from `@hatchet-dev/serverless/vercel`              | `ws` on the exported `http.Server`                                                         | Same limits; `vercel.json` rewrite so every route reaches the function                                                                                         |
| Node, self-hosted          | `createServer`, `nodeHandler` from `@hatchet-dev/serverless/node` | `ws` (optional peer)                                                                       | Your process lifetime and proxy; the operator dials https on 443 only, so terminate TLS in front                                                               |

`.` and `./cloudflare` import nothing from Node and run on any WinterCG runtime; `./vercel`
and `./node` import `node:http` and load `ws` (and `@vercel/functions`) at runtime. Every
adapter serves the same handler, so the healthcheck body, the signature checks and the
response mapping below are identical across runtimes; only the runtime name differs.

## Register the endpoint

The `hatchet serverless register` command is the intended way and is not built yet. Until it
lands, call the REST API directly. The tenant needs the serverless operator entitlement
(`tenant_entitlement.serverless_operator`); without it the API answers 403. The operator
registers the workflows under the names you declared, the way a worker does, so the `echo`
task above is triggered as `echo`. Two endpoints of one tenant may declare the same workflow;
both then serve it, like two workers.

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
never to the Worker. `kind` is `CLOUDFLARE_WORKERS` for a Worker and `GENERIC_HTTP` for a
Vercel or Node endpoint; the adapters apply their runtime's constraints themselves.

## Tests without Hatchet

`@hatchet-dev/serverless/testing` invokes a handler the way the operator would: it signs the
request and classifies the response with the operator's rules.

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
frame the endpoint sent, and `serverEvict()`, `sendError()`, `close()` and `closeStream()` to
act as the operator mid-flight; `op.emit(eventKey, payload)` satisfies a `waitForEvent`. The
fake enforces the operator's protocol rules (one ack-bearing request in flight, nothing after
the done frame, `done: evicted` only after an eviction ack, no durable request on a
non-durable task's socket, the 16-stream cap), so a violation fails the test.

A task that uses the client runs against whatever `client` you pass, so point it at a fake
engine (`createRouterTransport` from `@connectrpc/connect` as `transport`). `op.invoke` and
`op.deliver` take the socket for a task listed in `streams`, as the operator does, and the
test operator answers the run subscription the task opens: `op.completeRun(runId, [{ taskName,
output }])` finishes a child, or `runStream: { workflowRun: (runId) => event }` scripts the
engine.

## What the package verifies

Every request from the operator is signed with the endpoint's secret (`X-Hatchet-Signature`,
hex HMAC-SHA256). The handler checks, in this order:

- **POST bodies** (healthcheck, non-durable trigger): the HMAC over the raw body, then the
  `timestampUnixSeconds` field the signature covers, which must be within five minutes of the
  handler's clock in either direction (`REQUEST_MAX_AGE_SECONDS`), then the `endpointId` when the
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

On the invocation websocket (durable tasks, and non-durable tasks listed in `streams`) the
outcome travels in the final `done` frame: `{ output }` completes the task, `{ error, retry }`
fails it (retry is false for `NonRetryableError`, `ServerlessLimitationError` and after an
engine error frame), and `{ status: "evicted" }` after an eviction ack means the engine
re-invokes the task later. Every engine stream the task holds is closed before the done frame.

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
