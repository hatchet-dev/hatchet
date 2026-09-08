# @hatchet-dev/serverless

Serve Hatchet tasks from a serverless runtime without a worker process. Declare tasks with the
same API as the Hatchet TypeScript SDK, hand them to a runtime adapter, and register the
endpoint with Hatchet. The Hatchet serverless operator polls the endpoint for the workflows it
serves and delivers runs to it over signed HTTPS requests.

Status: preview, not yet published. This version serves non-durable tasks on Cloudflare Workers.
Durable tasks over the operator's websocket relay, the Vercel adapter, the
`hatchet serverless` CLI commands and the management module are the next phases.

## There is no Hatchet client in the serverless runtime

The package never reads a Hatchet API token and never instantiates a Hatchet client. Most
serverless runtimes cannot speak gRPC, and the transport that will replace it for them (a
ConnectRPC-style migration) is not decided. Everything a task needs arrives from the operator in
the signed request: input, parent outputs, retry count, metadata. Anything that would need a
client is unavailable and throws a `ServerlessLimitationError` naming the feature, which the
handler reports as a permanent failure (no retry, since retrying cannot help).

`ctx` members that throw `ServerlessLimitationError`:

| Member                                                                   | Reason                                                                                      |
| ------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------- |
| `ctx.runChild`, `ctx.runNoWaitChild`, `ctx.spawnWorkflow`                | child runs need a client                                                                    |
| `ctx.bulkRunChildren`, `ctx.bulkRunNoWaitChildren`, `ctx.spawnWorkflows` | same                                                                                        |
| `ctx.putStream`                                                          | streaming needs a client                                                                    |
| `ctx.cancel`                                                             | cancellation is the operator's: it drops the request and the task's `abortController` fires |
| `ctx.refreshTimeout`, `ctx.releaseSlot`                                  | there is no worker slot to release or timeout to extend                                     |
| `ctx.worker.labels`, `ctx.worker.upsertLabels`                           | there is no worker                                                                          |

`ctx` members that work unchanged: `input`, `parentOutput`, `retryCount`, `workflowRunId`,
`taskRunExternalId`, `workflowNameV1`, `taskName`, `additionalMetadata`, `triggers`, `errors`,
`priority`, `triggeringEventId`, `triggeringEventKey`, `abortController` and `cancelled`.
`ctx.logger.*` and `ctx.log` print to the console only; no log line reaches the engine.
`ctx.worker.id()` is `undefined` and `ctx.worker.hasWorkflow(name)` answers from the
declarations you handed to the adapter.

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

export const workflows = [echo, pipeline];
```

`hatchet` is the SDK's `task`, `durableTask`, `workflow` and `batchTask` factories bound to no
client. Durable tasks can be declared and are advertised to Hatchet, but this version cannot
serve them: the healthcheck reports `durable.supported: false`, so the operator never assigns
them to the endpoint.

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

`cloudflare({ workflows })` returns an `ExportedHandler<Env>` serving `POST /hatchet/healthcheck`
and `POST /hatchet/trigger`. It reads `HATCHET_SIGNING_SECRET` and the optional
`HATCHET_ENDPOINT_ID` from `env`. Every other path gets a 404 unless you pass a `fetch` option.

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

## How a trigger maps onto a response

The operator (`pkg/serverlessoperator/delivery.go`) reads the status code; a
`{"error", "retry"}` body overrides its message and retry decision.

| Task outcome                       | Response                                                              |
| ---------------------------------- | --------------------------------------------------------------------- |
| returns a value                    | `200`, the value as JSON                                              |
| returns `undefined`                | `204`                                                                 |
| throws `NonRetryableError`         | `422 {"error", "retry": false}`                                       |
| throws `ServerlessLimitationError` | `422 {"error", "retry": false}`                                       |
| throws anything else               | `500 {"error", "retry": true}`                                        |
| action not served here             | `404 {"error", "retry": false}`                                       |
| durable action sent as a POST      | `422 {"error", "retry": false}`                                       |
| bad signature                      | `401 {"error", "retry": false}`                                       |
| durable websocket upgrade          | `426 {"error", "retry": false}` (relay not available in this version) |

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
