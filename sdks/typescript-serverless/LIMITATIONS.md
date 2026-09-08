# Limitations

What a task cannot do when it runs on a serverless endpoint, and why.

## No Hatchet client, no token

The serverless runtime holds no Hatchet client and never reads a Hatchet API token. Most
serverless runtimes cannot speak gRPC, which is what the SDK's client uses, and the transport
that will replace it for them is not decided. The package therefore limits itself to what the
operator sends over the signed request. Any `ctx` member that would need a client throws
`ServerlessLimitationError` with the feature in its message; the trigger handler answers such a
failure with `422` and `retry: false`, because retrying cannot help.

| Member                                                                                                                              | Behaviour                                                                                      |
| ----------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- |
| `ctx.runChild`, `ctx.runNoWaitChild`, `ctx.spawnWorkflow`, `ctx.bulkRunChildren`, `ctx.bulkRunNoWaitChildren`, `ctx.spawnWorkflows` | throw                                                                                          |
| `ctx.putStream`                                                                                                                     | throws                                                                                         |
| `ctx.cancel`                                                                                                                        | throws; the operator cancels a run by dropping the request, which aborts `ctx.abortController` |
| `ctx.refreshTimeout`, `ctx.releaseSlot`                                                                                             | throw                                                                                          |
| `ctx.worker.labels()`, `ctx.worker.upsertLabels()`                                                                                  | throw                                                                                          |
| `ctx.worker.id()`                                                                                                                   | `undefined`                                                                                    |
| `ctx.worker.hasWorkflow(name)`                                                                                                      | answers from the declarations passed to the adapter                                            |
| `ctx.logger.*`, `ctx.log`                                                                                                           | console only; nothing reaches the engine                                                       |
| `declaration.run`, `.runNoWait`, `.schedule`, `.cron`, `.delay`, `.metrics`, `.get`                                                 | throw (the declaration has no client)                                                          |

Everything else on `Context` is pure and works unchanged: `input`, `parentOutput`, `retryCount`,
`workflowRunId`, `taskRunExternalId`, `workflowNameV1`, `taskName`, `additionalMetadata`,
`triggers`, `filterPayload`, `errors`, `priority`, `triggeringEventId`, `triggeringEventKey`,
`childIndex`, `childKey`, `parentWorkflowRunId`, `abortController`, `cancelled`.

The next phase lifts part of this for durable tasks: `spawnChild`, `sleepFor`, `waitFor`,
`waitForEvent` and `now` will run over the operator's websocket relay, which carries
`DurableTaskRequest` frames to the engine on the task's behalf. Non-durable tasks keep the
limitation.

## Durable tasks are not served yet

Durable declarations are advertised in the healthcheck (`isDurable: true`) so the registration
matches the one a worker would produce, but this version reports `durable.supported: false`
and the operator never assigns a durable task to the endpoint. A durable action delivered as a
POST anyway is answered `422` with `retry: false`; a websocket upgrade is answered `426`.

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

No slots, heartbeats, long-lived connections, health server or `process.on`. The operator owns
liveness (the healthcheck poll), routing and retries. A task's wall-clock budget is the runtime's
(`limits.cpu_ms` on Cloudflare) and the endpoint's `requestTimeoutSeconds` in Hatchet, whichever
ends first.

## `serve` subsets before shared namespaces

`serve` limits the action ids listed in the healthcheck and answered by the trigger route. With
today's operator each endpoint has its own namespace and registers every advertised workflow in
full, so a task outside the subset is still registered under the endpoint's namespace and, if
triggered, fails with `404` from the endpoint. Splitting one workflow across endpoints needs the
operator change that lets a namespace span endpoints.

## Response size

The operator reads at most 4 MiB of a trigger response. Larger outputs fail the task without
retry.
