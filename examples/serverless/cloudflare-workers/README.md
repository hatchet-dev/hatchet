# Hatchet serverless endpoint on Cloudflare Workers

A Worker that serves two Hatchet workflows without running a worker process. The Hatchet
serverless operator polls it for the workflows it serves and delivers assigned tasks to it
over signed HTTPS requests; durable tasks run over a websocket the operator dials.

- `echo`: a non-durable task. The operator POSTs the task, the Worker returns its input.
- `sleep-then-echo`: a durable task. It memoizes a timestamp, sleeps 3 seconds and returns both.
  The sleep is longer than the endpoint's inline wait budget, so the first invocation evicts
  itself and the engine re-invokes the task when the sleep is over; the second invocation gets
  the memoized timestamp from the log and finishes.

## Layout

| File | Purpose |
|---|---|
| `src/hatchet.ts` | Dependency-free shim for the endpoint contract: signature verification, namespace handling, the healthcheck body, and `DurableClient` (memo, sleep, wait_for, evict, done) over the websocket protocol. |
| `src/index.ts` | The fetch handler: `POST /hatchet/healthcheck`, `POST /hatchet/trigger` (non-durable), websocket upgrade on `/hatchet/trigger` (durable). |
| `wrangler.toml` | Worker config. `limits.cpu_ms = 300000` raises the CPU budget to the 5 minute maximum (Workers Paid plan). |

The wire contract is `api-contracts/v1/serverless.proto` (every request, response and websocket
frame, carried as protojson), plus `pkg/serverlessoperator/contract/http.go` (headers, the upgrade
signature payload) and `pkg/serverlessoperator/durable/protocol.go` (websocket close codes).
`src/hatchet.ts` mirrors it by hand so the example stays dependency-free; the
`@hatchet-dev/serverless` package generates the same types from the proto.

## Deploy

Prerequisites: Node 20+, pnpm (or npm), a Cloudflare account on the Workers Paid plan (needed for
`cpu_ms` above 30 s; remove the `[limits]` block to try it on the Free plan, where the 10 ms CPU
limit is enough for `echo` but may not be for anything real), and `wrangler login`.

```sh
pnpm install

# The secret that signs every request the operator sends. 32+ characters. Keep the value: it
# is the endpoint's signingSecret when you register it in Hatchet.
openssl rand -hex 32
pnpm wrangler secret put HATCHET_SIGNING_SECRET   # paste the value at the prompt

pnpm run deploy                                    # prints https://<name>.<account>.workers.dev
```

Then create the endpoint in Hatchet with `kind: CLOUDFLARE_WORKERS`,
`healthcheckUrl: https://<worker>/hatchet/healthcheck`, `triggerUrl: https://<worker>/hatchet/trigger`
and the same `signingSecret`. The response carries the endpoint's `namespace`; workflows are registered
as `<namespace>_echo` and `<namespace>_sleep-then-echo`.

Optional: `pnpm wrangler secret put HATCHET_ENDPOINT_ID` with the endpoint id makes the Worker refuse
requests and websocket upgrades carrying another endpoint id.

## What the Worker verifies

Every request from the operator is signed with the endpoint's secret (`X-Hatchet-Signature`,
hex HMAC-SHA256). The Worker checks, in this order:

- **POST bodies** (healthcheck, non-durable trigger): the HMAC over the raw body, then the
  `timestamp` field the signature covers, which must be within five minutes of the Worker's
  clock in either direction (`REQUEST_MAX_AGE_SECONDS`), then the `endpointId` when
  `HATCHET_ENDPOINT_ID` is set. A captured request is therefore only replayable for five
  minutes; a task with side effects should treat `(endpointId, taskRunExternalId, retryCount)`
  as its idempotency key.
- **Websocket upgrades** (durable trigger): the HMAC over
  `endpointId.timestamp.nonce.taskId.invocation` (so a signature made for one endpoint is no
  good at another that shares the secret), the same five minute window on
  `X-Hatchet-Timestamp`, and the nonce, which is consumed from an in-memory set after the
  signature verified (`NonceSet`: every accepted nonce is kept until its window has passed,
  and a set holding 4096 live nonces refuses further upgrades with 503 rather than forget
  one). That set is per isolate; a production endpoint should consume nonces in a Durable
  Object or an expiring KV key so a replay that lands in another isolate is caught.
- **The first frame**: its task id and invocation must match the verified upgrade headers,
  otherwise the socket is closed with 1008 before any task code runs.

## Watch it run

```sh
pnpm run tail
```

shows the healthcheck polls, each trigger, each accepted upgrade and the durable task's memo, wait,
eviction and done steps.

## Local development

`pnpm run dev` serves the Worker on `http://localhost:8787`. The operator only dials `https` URLs on
port 443, so a local Worker needs either a tunnel (`cloudflared tunnel`) or the operator's
`SERVERLESS_OPERATOR_INSECURE_DESTINATIONS=true` development switch, which allows `http://localhost:8787`
and rewrites it to `ws://` for durable tasks. Put the secret in `.dev.vars` (`HATCHET_SIGNING_SECRET=...`)
for `wrangler dev`.
