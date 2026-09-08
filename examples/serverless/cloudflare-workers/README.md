# Hatchet serverless endpoint on Cloudflare Workers

A Worker that serves Hatchet tasks without running a worker process, built on
`@hatchet-dev/serverless`. The Hatchet serverless operator polls it for the workflows it serves
and delivers assigned tasks to it over signed HTTPS requests.

- `echo`: a non-durable task. The operator POSTs the task, the Worker returns the message with
  the run id and retry count.
- `sleep-then-echo`: a durable task. It records a timestamp, sleeps 3 seconds and returns both.
  It is declared and advertised, but not runnable yet: this version of `@hatchet-dev/serverless`
  reports `durable.supported: false`, so the operator never assigns it. The durable relay is the
  package's next phase; nothing in the example changes when it lands.

## Layout

| File | Purpose |
|---|---|
| `src/tasks.ts` | The declarations, written with `hatchet.task` and `hatchet.durableTask` from `@hatchet-dev/serverless`. The same file works on a regular worker. |
| `src/index.ts` | `export default cloudflare({ workflows })`: the adapter serves `POST /hatchet/healthcheck` and `POST /hatchet/trigger` and reads the secret from `env.HATCHET_SIGNING_SECRET`. |
| `wrangler.toml` | Worker config. `limits.cpu_ms = 300000` raises the CPU budget to the 5 minute maximum (Workers Paid plan). |

The wire contract is `api-contracts/v1/serverless.proto` (every request and response, carried as
protojson) plus `pkg/serverlessoperator/contract/http.go` (headers, the signature scheme). The
package generates its types from the proto, so nothing here is hand-written.

## Build the package first

`@hatchet-dev/serverless` is not published yet; this example links it from the repository, and
the package links the TypeScript SDK from its build output. From the repository root:

```sh
cd sdks/typescript && pnpm install && pnpm tsc:build && pnpm prepublish
cd ../typescript-serverless && pnpm install && pnpm build
cd ../../examples/serverless/cloudflare-workers && pnpm install
pnpm run typecheck
```

## Deploy

Prerequisites: Node 22, pnpm, a Cloudflare account on the Workers Paid plan (needed for `cpu_ms`
above 30 s; remove the `[limits]` block to try it on the Free plan, where the 10 ms CPU limit is
enough for `echo` but may not be for anything real), and `wrangler login`.

```sh
# The secret that signs every request the operator sends. 32+ characters. Keep the value: it
# is the endpoint's signingSecret when you register it in Hatchet.
openssl rand -hex 32
pnpm wrangler secret put HATCHET_SIGNING_SECRET   # paste the value at the prompt

pnpm run deploy                                    # prints https://<name>.<account>.workers.dev
```

Then create the endpoint in Hatchet with `kind: CLOUDFLARE_WORKERS`,
`healthcheckUrl: https://<worker>/hatchet/healthcheck`, `triggerUrl: https://<worker>/hatchet/trigger`
and the same `signingSecret` (`POST /api/v1/stable/tenants/{tenant}/serverless/endpoints`; the
`hatchet serverless register` command will do this later). The response carries the endpoint's
`namespace`; workflows are registered as `<namespace>_echo` and `<namespace>_sleep-then-echo`.

Trigger `echo` with the input `{"message": "hello"}`; it returns
`{"echo": "hello", "workflowRunId": "...", "retryCount": 0}`.

Optional: `pnpm wrangler secret put HATCHET_ENDPOINT_ID` with the endpoint id makes the Worker
refuse durable upgrades carrying another endpoint id once the relay exists.

## Watch it run

```sh
pnpm run tail
```

shows each request. The adapter logs a task's `ctx.logger` calls and failed tasks; healthchecks
and successful triggers are silent.

## Local development

`pnpm run dev` serves the Worker on `http://localhost:8787`. The operator only dials `https` URLs on
port 443, so a local Worker needs either a tunnel (`cloudflared tunnel`) or the operator's
`SERVERLESS_OPERATOR_INSECURE_DESTINATIONS=true` development switch. Put the secret in `.dev.vars`
(`HATCHET_SIGNING_SECRET=...`) for `wrangler dev`.

Without Hatchet at all, `@hatchet-dev/serverless/testing` invokes the tasks the way the operator
would; see the package README.
