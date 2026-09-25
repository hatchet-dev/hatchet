# Limitations

What a task cannot do when it runs on a serverless endpoint, and why.

## No Hatchet client, no token

The serverless runtime holds no Hatchet client and never reads a Hatchet API token. Most
serverless runtimes cannot speak gRPC, which is what the SDK's client uses, and the transport
that will replace it for them is not decided. The package therefore limits itself to what the
operator sends over the signed request or relays over the durable websocket. Any `ctx` member
that would need a client throws `ServerlessLimitationError` with the feature in its message; the
trigger handler answers such a failure with `422` and `retry: false`, because retrying cannot
help.

| Member                                                                                                                              | Behaviour                                                                                                            |
| ----------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------- |
| `ctx.runChild`, `ctx.runNoWaitChild`, `ctx.spawnWorkflow`, `ctx.bulkRunChildren`, `ctx.bulkRunNoWaitChildren`, `ctx.spawnWorkflows` | throw; durable tasks use `ctx.spawnChild` and `ctx.spawnChildren`, which run over the relay                          |
| `ctx.putStream`                                                                                                                     | throws                                                                                                               |
| `ctx.cancel`                                                                                                                        | throws; the operator cancels a run by dropping the request or closing the socket, which aborts `ctx.abortController` |
| `ctx.refreshTimeout`, `ctx.releaseSlot`                                                                                             | throw                                                                                                                |
| `ctx.worker.labels()`, `ctx.worker.upsertLabels()`                                                                                  | throw                                                                                                                |
| `ctx.worker.id()`                                                                                                                   | `undefined`                                                                                                          |
| `ctx.worker.hasWorkflow(name)`                                                                                                      | answers from the declarations passed to the adapter                                                                  |
| `ctx.logger.*`, `ctx.log`                                                                                                           | console only; nothing reaches the engine                                                                             |
| `declaration.run`, `.runNoWait`, `.schedule`, `.cron`, `.delay`, `.metrics`, `.get`                                                 | throw (the declaration has no client)                                                                                |

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

No slots, heartbeats, long-lived connections other than a durable invocation's socket, health
server or `process.on`. The operator owns liveness (the healthcheck poll), routing and retries. A
non-durable task's wall-clock budget is the runtime's (`limits.cpu_ms` on Cloudflare) and the
endpoint's `requestTimeoutSeconds` in Hatchet, whichever ends first.

## `serve` subsets before shared namespaces

`serve` limits the action ids listed in the healthcheck and answered by the trigger route. With
today's operator each endpoint has its own namespace and registers every advertised workflow in
full, so a task outside the subset is still registered under the endpoint's namespace and, if
triggered, fails with `404` from the endpoint. Splitting one workflow across endpoints needs the
operator change that lets a namespace span endpoints.

## Response size

The operator reads at most 4 MiB of a trigger response and of a durable frame. Larger outputs
fail the task without retry.
