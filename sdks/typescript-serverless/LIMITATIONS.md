# Limitations

What a task can and cannot do when it runs on a serverless endpoint, and why.

## The client and the token

A serverless endpoint holds no long-lived connection to the engine. The handler takes an
optional `client`, the SDK's core client (`HatchetCore` from `@hatchet-dev/typescript-sdk/core`),
which makes unary Connect calls over `fetch`, so it runs wherever the handler runs. Its token
lives in the platform's secret store next to the signing secret (`HATCHET_CLIENT_TOKEN` on
Cloudflare); the operator never sends a token to the endpoint. The token is tenant-wide, so a
task holding it may trigger, cancel and observe any run of the tenant; namespace-scoped tokens
are a later step.

Anything a task does that reaches the engine goes one of two ways:

- **Unary calls** (`ctx.runNoWaitChild`, `ctx.putStream`, `ctx.log`, `ctx.cancel`, and the
  trigger behind `ctx.runChild`) go from the endpoint to the engine through the client.
- **Streams** (awaiting a child's result, reading a durable stream topic, following a run's
  events) go through the operator over the invocation websocket: the task sends a
  `stream_open` frame naming the procedure (`/Dispatcher/SubscribeToWorkflowRuns` for a
  child's result, `/v1.V1Streams/Subscribe` for a topic), the operator opens the engine
  stream with its own credentials and relays its messages back. No stream is ever opened
  from the endpoint to the engine, and the operator allows only
  `/Dispatcher/SubscribeToWorkflowRuns`, `/Dispatcher/SubscribeToWorkflowEvents`,
  `/v1.V1Dispatcher/ListenForDurableEvent` and `/v1.V1Streams/Subscribe`, at most 16 open
  streams per socket. The package keeps every child of a task on one
  `SubscribeToWorkflowRuns` stream, so a fan-out takes one slot however many children it
  awaits; each `clientFromContext(ctx).streams.events(topic)` or `runs.events(runId)`
  iteration holds one slot while it runs and frees it when the task stops iterating.

Without a `client`, every member that needs one throws `ServerlessLimitationError` with the
feature in its message and "configure `client` to enable this"; the trigger handler answers
such a failure with `422` and `retry: false`, because retrying cannot help. Existing
deployments without a client are unchanged.

### Pure tasks and the invocation socket

The operator dials the invocation websocket for every task by default: every durable task, and
every non-durable task except those declared with `hatchetClient: false`
(`hatchet.task({ name, fn, hatchetClient: false })`, or the same option on a `workflow.task(...)`
entry). The option is a statement about the task: it is a pure function of its input and never
talks to Hatchet at runtime. The healthcheck lists the socket tasks as
`tasks: [{ action, streams: true }]`, the proto field the operator routes by; a pure task is
delivered over a signed POST, which is cheaper (no upgrade, no socket held for the task's
lifetime, no slot in the relay's stream table) and has nothing to await on. So:

- A task that talks to Hatchet (`ctx.runChild`, `ctx.runNoWaitChild`, `ctx.putStream`,
  `ctx.cancel`, `clientFromContext(ctx)`, a log line written to the engine) keeps the default.
  It runs on the socket, waits for its children inline and never evicts.
- A pure task runs under a context whose client-backed members throw
  `ServerlessLimitationError` naming `hatchetClient: false`, whether or not the handler has a
  `client`; `ctx.log` and `ctx.logger.*` print to the console only. Everything that comes with
  the request (`input`, `parentOutput`, `retryCount`, the run ids) works as in any task.
- A durable task always talks to Hatchet: its sleeps, waits and child runs go through the
  durable log over the socket. `hatchetClient: false` on `hatchet.durableTask` or
  `workflow.durableTask` throws when the task is declared.
- A task that keeps the default but arrives over a POST anyway (an operator that predates
  stream delivery, or an endpoint whose adapter cannot accept the upgrade: `durable: false`,
  Vercel without Fluid compute, Node without `ws`) still runs. Its triggers happen, and only a
  wait on a child's result (`ctx.runChild`, `ctx.bulkRunChildren`, or `result()` on a reference
  from `ctx.runNoWaitChild`) and a read over the socket (`clientFromContext(ctx).streams.events`,
  `clientFromContext(ctx).runs.events`) throw `ServerlessLimitationError` saying the task was
  invoked over a POST. An adapter without the upgrade lists no stream task in its healthcheck
  on purpose: POSTing everything fails only the waits, where listing the tasks would fail every
  invocation at the upgrade.
- A reference created by a `HatchetCore` you hold yourself (`hatchet.run(...)` inside or
  outside a task) knows nothing about the socket and polls `GetRunDetails`, the way `/core`
  does everywhere.
- Inside a durable task, `ctx.runChild` and its variants throw: a durable task spawns children
  with `ctx.spawnChild` and `ctx.spawnChildren`, which record the child in the durable event
  log and replay its result after an eviction. `ctx.runChild` would trigger the child again on
  every re-invocation.

Why the option is declared rather than inferred: the handler cannot see inside the function,
and the operator chooses the delivery before the request is sent (a POST cannot be upgraded
mid-flight), so the task says what it is, next to its code. Changing the option is a catalog
change: the next healthcheck (every `pollIntervalSeconds`) re-registers the task and the
operator switches delivery.

| Member                                                                                       | Without `client`, or `hatchetClient: false`                                                                          | With `client`                                                                                                      |
| -------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------ |
| `ctx.runNoWaitChild`, `ctx.bulkRunNoWaitChildren`, `ctx.spawnWorkflow`, `ctx.spawnWorkflows` | throw                                                                                                                | trigger through the client; the reference's `result()` needs the socket the task has by default                    |
| `ctx.runChild`, `ctx.bulkRunChildren`                                                        | throw                                                                                                                | trigger through the client, await on the socket the task has by default                                            |
| any of the above in a durable task                                                           | throw                                                                                                                | throw; use `ctx.spawnChild` and `ctx.spawnChildren`, which go through the durable log                              |
| `ctx.putStream`                                                                              | throws                                                                                                               | `PutStreamEvent` through the client                                                                                |
| `ctx.log`, `ctx.logger.*`                                                                    | console only                                                                                                         | console, and `ctx.log` also writes the line to the engine (a failed write is a console warning)                    |
| `ctx.cancel`                                                                                 | throws; the operator cancels a run by dropping the request or closing the socket, which aborts `ctx.abortController` | `CancelTasks` through the client                                                                                   |
| `clientFromContext(ctx)`                                                                     | throws                                                                                                               | the handler's `HatchetCore`, with the two socket-backed reads below                                                |
| `clientFromContext(ctx).streams.events(topic)`, `clientFromContext(ctx).runs.events(runId)`  | throw                                                                                                                | the engine stream relayed over the socket the task has by default; over a POST they throw                          |
| `ctx.refreshTimeout`                                                                         | throws                                                                                                               | `RefreshTimeout` through the client; it extends the engine's timeout of the run, not the adapter's request timeout |
| `ctx.releaseSlot`                                                                            | throws                                                                                                               | `ReleaseSlot` through the client; the run stops counting against the tenant's serverless worker slots              |
| `ctx.worker.labels()`, `ctx.worker.upsertLabels()`                                           | throw                                                                                                                | throw; there is no worker                                                                                          |
| `ctx.worker.id()`                                                                            | `undefined`                                                                                                          | `undefined`                                                                                                        |
| `ctx.worker.hasWorkflow(name)`                                                               | answers from the declarations passed to the adapter                                                                  | same                                                                                                               |
| `declaration.run`, `.runNoWait`, `.schedule`, `.cron`, `.delay`, `.metrics`, `.get`          | throw (the declaration has no client)                                                                                | throw; call `client.run(declaration, input)` on a `HatchetCore` instead                                            |

Everything else on `Context` is pure and works unchanged: `input`, `parentOutput`, `retryCount`,
`workflowRunId`, `taskRunExternalId`, `workflowNameV1`, `taskName`, `additionalMetadata`,
`triggers`, `filterPayload`, `errors`, `priority`, `triggeringEventId`, `triggeringEventKey`,
`childIndex`, `childKey`, `parentWorkflowRunId`, `abortController`, `cancelled`.

In a durable task, `ctx.now`, `ctx.sleepFor`, `ctx.sleepUntil`, `ctx.waitFor`, `ctx.waitForEvent`,
`ctx.spawnChild` and `ctx.spawnChildren` work: they become durable event requests the operator
relays to the engine over the invocation's websocket, and their results replay from the engine's
event log on re-invocation.

### Reading a durable stream from a task

`clientFromContext(ctx).streams.events(topic, options)` is the Node client's
`hatchet.streams.events`: an `AsyncIterable` of `{ payload, cursor, createdAt }` with the same
`namespace`, `cursor` and `signal` options. The core client cannot carry it (a server stream
over a unary transport), so inside a task it goes over the invocation socket: one
`/v1.V1Streams/Subscribe` engine stream per iteration, relayed by the operator. The differences
from the Node client:

- It needs an operator that relays the subscription. An older operator closes the stream with
  code 12 (unimplemented) and the iteration throws `StreamClosedError` saying so; a tenant
  without the `durable_streams` entitlement sees code 7 (permission denied) the same way.
- It does not resubscribe when the stream drops: the iteration ends on the engine finishing or
  hanging up the stream, on `options.signal`, or when the task stops iterating (`break`,
  `return()`), and throws `StreamClosedError` on any other close, carrying the `cursor` of the
  last message yielded so the task can resume from it.
- A message whose payload was uploaded (over the 4 MiB inline limit, which only the Node client
  publishes) cannot be fetched, since the core client has no REST path: the iteration throws
  a `HatchetError` naming the message.
- `signal` is the way to stop a read that is waiting for the next message; `return()` on the
  iterator takes effect between messages, as with any async generator.

`clientFromContext(ctx).runs.events(runId)` reads `/Dispatcher/SubscribeToWorkflowEvents` the
same way, yielding each `WorkflowEvent` until the engine's hang-up event.

## Durable tasks and the inline wait budget

A durable invocation lives on a websocket the operator dials. It may wait inline for
`inlineWaitBudgetMs` (a setting of the endpoint in Hatchet, 5000 by default) for a sleep, event
or child run to complete; past the budget it evicts itself and the engine re-invokes the task
when the awaited entry is satisfied. The declaration's `evictionPolicy` is ignored: the budget is
the only eviction rule. A re-invocation re-executes the function from the top, so it must be
deterministic; memoized calls (`ctx.now`) replay, and a replay that emits different events than
the log recorded fails with a nondeterminism error and no retry.

The runtime's wall-clock and CPU limits apply to each invocation, not to the run: a durable task
that sleeps for a day spends that day evicted, not on the socket. Child runs spawned from a
durable task must be served by an endpoint or worker of their own; the relay only carries the
request and the result.

A non-durable task on a socket (every one not declared with `hatchetClient: false`) has no
budget and never evicts: it waits for its children inline, bounded by the runtime's limits and
the endpoint's `requestTimeoutSeconds`, and is always invocation 1.

## Declaration options that are ignored

Options that need a worker are removed from the registration. Each produces one `console.warn`
when the handler is built, listing where it was found, and the declaration object itself is not
modified (a worker elsewhere may register the same declaration).

| Option                                             | Why it is ignored                                                                       |
| -------------------------------------------------- | --------------------------------------------------------------------------------------- |
| `slotCost`, `slotRequests`                         | a serverless endpoint has no worker slots                                               |
| `batch`                                            | batch tasks are not supported; the task is registered as a plain task                   |
| `desiredWorkerLabels`, `taskDefaults.workerLabels` | worker labels do not apply; the operator routes by action id                            |
| `sticky` (workflow level)                          | sticky assignment needs a worker                                                        |
| `evictionPolicy`                                   | the endpoint's `inlineWaitBudgetMs` in Hatchet decides when a durable invocation evicts |

## No worker semantics

No slots, heartbeats, long-lived connections other than an invocation's socket, health server
or `process.on`. The operator owns liveness (the healthcheck poll), routing and retries. A
non-durable task's wall-clock budget is the runtime's (`limits.cpu_ms` on Cloudflare) and the
endpoint's `requestTimeoutSeconds` in Hatchet, whichever ends first.

## `serve` subsets

`serve` limits the action ids listed in the healthcheck and answered by the trigger route. The
operator registers every advertised workflow in full and derives the endpoint's actions from
the workflow definitions, so a task outside the subset is still registered for the endpoint and,
if triggered, fails with `404` from the endpoint. Two endpoints that declare the same workflow
both register all of its actions, so splitting one workflow across endpoints is not supported.
Socket tasks outside `serve` are not listed under `tasks`.

## Response size

The operator reads at most 4 MiB of a trigger response and of a durable frame. Larger outputs
fail the task without retry. The same frame limit applies to every stream frame, including the
terminal event of an awaited child and each batch of a topic's messages, so a child whose
outputs come to more than 4 MiB cannot be awaited over the socket.
