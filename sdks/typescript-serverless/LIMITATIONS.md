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
- **Streams** (awaiting a child's result) go through the operator over the invocation
  websocket: the task sends a `stream_open` frame naming `/Dispatcher/SubscribeToWorkflowRuns`,
  the operator opens the engine stream with its own credentials and relays the run's terminal
  event back. No stream is ever opened from the endpoint to the engine, and the operator
  allows only `/Dispatcher/SubscribeToWorkflowRuns`, `/Dispatcher/SubscribeToWorkflowEvents`
  and `/v1.V1Dispatcher/ListenForDurableEvent`, at most 16 open streams per socket. The
  package keeps every child of a task on one `SubscribeToWorkflowRuns` stream, so a fan-out
  takes one slot however many children it awaits.

Without a `client`, every member that needs one throws `ServerlessLimitationError` with the
feature in its message and "configure `client` to enable this"; the trigger handler answers
such a failure with `422` and `retry: false`, because retrying cannot help. Existing
deployments without a client are unchanged.

### The `streams` rule

A socket exists only for the tasks the endpoint's catalog asks one for: every durable task,
and every non-durable task listed in the handler's `streams` option (`streams: [parent]` or
`streams: ['pipeline:summarize']`), which the healthcheck advertises as
`tasks: [{ action, streams: true }]`. Every other task is delivered over a signed POST, which is
cheaper and has nothing to await on. So:

- A non-durable task that awaits a child (`ctx.runChild`, `ctx.bulkRunChildren`, or `result()`
  on a reference from `ctx.runNoWaitChild`) must be listed in `streams`. Over a POST the
  trigger still happens, but the wait throws `ServerlessLimitationError` naming the option.
- A reference created by a `HatchetCore` you hold yourself (`hatchet.run(...)` inside or
  outside a task) knows nothing about the socket and polls `GetRunDetails`, the way `/core`
  does everywhere.
- Inside a durable task, `ctx.runChild` and its variants throw: a durable task spawns children
  with `ctx.spawnChild` and `ctx.spawnChildren`, which record the child in the durable event
  log and replay its result after an eviction. `ctx.runChild` would trigger the child again on
  every re-invocation.

| Member                                                                                       | Without `client`                                                                                                     | With `client`                                                                                   |
| -------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------- |
| `ctx.runNoWaitChild`, `ctx.bulkRunNoWaitChildren`, `ctx.spawnWorkflow`, `ctx.spawnWorkflows` | throw                                                                                                                | trigger through the client; the reference's `result()` needs the socket (`streams`)             |
| `ctx.runChild`, `ctx.bulkRunChildren`                                                        | throw                                                                                                                | trigger through the client, await on the socket; the task must be listed in `streams`           |
| any of the above in a durable task                                                           | throw                                                                                                                | throw; use `ctx.spawnChild` and `ctx.spawnChildren`, which go through the durable log           |
| `ctx.putStream`                                                                              | throws                                                                                                               | `PutStreamEvent` through the client                                                             |
| `ctx.log`, `ctx.logger.*`                                                                    | console only                                                                                                         | console, and `ctx.log` also writes the line to the engine (a failed write is a console warning) |
| `ctx.cancel`                                                                                 | throws; the operator cancels a run by dropping the request or closing the socket, which aborts `ctx.abortController` | `CancelTasks` through the client                                                                |
| `ctx.refreshTimeout`                                                                         | throws                                                                                                               | throws; set the task's `executionTimeout` and the endpoint's `requestTimeoutSeconds` instead    |
| `ctx.releaseSlot`, `ctx.worker.labels()`, `ctx.worker.upsertLabels()`                        | throw                                                                                                                | throw; there is no worker                                                                       |
| `ctx.worker.id()`                                                                            | `undefined`                                                                                                          | `undefined`                                                                                     |
| `ctx.worker.hasWorkflow(name)`                                                               | answers from the declarations passed to the adapter                                                                  | same                                                                                            |
| `declaration.run`, `.runNoWait`, `.schedule`, `.cron`, `.delay`, `.metrics`, `.get`          | throw (the declaration has no client)                                                                                | throw; call `client.run(declaration, input)` on a `HatchetCore` instead                         |

Everything else on `Context` is pure and works unchanged: `input`, `parentOutput`, `retryCount`,
`workflowRunId`, `taskRunExternalId`, `workflowNameV1`, `taskName`, `additionalMetadata`,
`triggers`, `filterPayload`, `errors`, `priority`, `triggeringEventId`, `triggeringEventKey`,
`childIndex`, `childKey`, `parentWorkflowRunId`, `abortController`, `cancelled`.

In a durable task, `ctx.now`, `ctx.sleepFor`, `ctx.sleepUntil`, `ctx.waitFor`, `ctx.waitForEvent`,
`ctx.spawnChild` and `ctx.spawnChildren` work: they become durable event requests the operator
relays to the engine over the invocation's websocket, and their results replay from the engine's
event log on re-invocation.

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

A non-durable task on a socket (one listed in `streams`) has no budget and never evicts: it
waits for its children inline, bounded by the runtime's limits and the endpoint's
`requestTimeoutSeconds`, and is always invocation 1.

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
`streams` entries outside `serve` are not advertised.

## Response size

The operator reads at most 4 MiB of a trigger response and of a durable frame. Larger outputs
fail the task without retry. The same frame limit applies to every stream frame, including the
terminal event of an awaited child, so a child whose outputs come to more than 4 MiB cannot be
awaited over the socket.
