# Hatchet serverless endpoint on Cloudflare Workers

A Worker that serves Hatchet tasks without running a worker process, built on
`@hatchet-dev/serverless`. The Hatchet serverless operator polls it for the workflows it serves
and delivers assigned tasks to it over signed HTTPS requests.

- `echo`: a non-durable task. The operator POSTs the task, the Worker returns the message with
  the run id and retry count.
- `sleep-then-echo`: a durable task. It records a timestamp, sleeps 3 seconds and returns both.
  The sleep is longer than the endpoint's inline wait budget, so the first invocation evicts
  itself and the engine re-invokes the task when the sleep is over; the second invocation gets
  the memoized timestamp from the event log and finishes with `invocation: 2`.

The operator registers the workflows under the names declared here, the way a worker would, so
`echo` is triggered as `echo` and its action id is `echo:echo`.

## Layout

| File | Purpose |
|---|---|
| `src/tasks.ts` | The declarations, written with `hatchet.task` and `hatchet.durableTask` from `@hatchet-dev/serverless`. The same file works on a regular worker. |
| `src/index.ts` | `export default cloudflare({ workflows })`: the adapter serves `POST /hatchet/healthcheck`, `POST /hatchet/trigger` and the durable websocket upgrade on `/hatchet/trigger`, and reads the secret from `env.HATCHET_SIGNING_SECRET`. |
| `wrangler.toml` | Worker config. `limits.cpu_ms = 300000` raises the CPU budget to the 5 minute maximum (Workers Paid plan). |

The wire contract is `api-contracts/v1/serverless.proto` (every request, response and websocket
frame, carried as protojson) plus `pkg/serverlessoperator/contract/http.go` (headers, the
signature scheme) and `pkg/serverlessoperator/durable/protocol.go` (close codes). The package
generates its types from the proto, so nothing here is hand-written.

## Build the package first

`@hatchet-dev/serverless` is not published yet; this example links it from the repository, and
the package links the TypeScript SDK from its build output. From the repository root:

```sh
cd sdks/typescript && pnpm install && pnpm tsc:build && pnpm prepublish
cd ../typescript-serverless && pnpm install && pnpm build
cd ../../examples/serverless/cloudflare-workers && pnpm install
pnpm run typecheck
```

## Secrets: what lives where

The Worker holds exactly one secret, `HATCHET_SIGNING_SECRET`, the HMAC key the operator signs
every request with. It never holds a Hatchet API token: the package has no Hatchet client, and
the operator never sends a token to the Worker. The API token stays in the shell (or deploy
script) that registers the endpoint and triggers runs.

- `wrangler dev` reads the secret from `.dev.vars` (gitignored).
- `wrangler deploy` reads it from the Worker secret set with `wrangler secret put`.

The same value is the endpoint's `signingSecret` when it is registered in Hatchet. It must be
32 characters or longer; `openssl rand -hex 32` makes a fitting one.

## Run against a local engine

### What you need

- A Hatchet engine and API built from this repository with the in-engine serverless operator
  on: `SERVER_SERVERLESS_OPERATOR_ENABLED=true` and
  `SERVER_SERVERLESS_OPERATOR_ALLOW_EMPTY_INFRA_CIDRS=true` (the second lets the operator start
  without an infrastructure CIDR denylist; both are documented in
  `frontend/docs/content/docs/self-hosting/configuration-options.mdx`). The commands below
  assume the API at `http://localhost:8888` and gRPC at `localhost:7070` (the engine's default
  ports) with gRPC served without TLS (`SERVER_GRPC_INSECURE=true`).
- An API token for the tenant (dashboard: Settings, API Tokens) and the tenant id.
- `psql` access to the engine database, `jq`, `curl`, `openssl`, and `cloudflared`
  (Cloudflare's tunnel client) for section 3.

```sh
export API=http://localhost:8888
export TENANT_ID=<tenant id>
export HATCHET_CLIENT_TOKEN=<api token>
export DATABASE_URL=<the engine's postgres URL>
curl -sf -H "Authorization: Bearer $HATCHET_CLIENT_TOKEN" $API/api/v1/tenants/$TENANT_ID >/dev/null && echo token ok
```

### 1. Entitle the tenant

Endpoint creation answers `403 the serverless operator is not enabled for this tenant` until
`tenant_entitlement.serverless_operator` is true for the tenant
(`api/v1/server/handlers/v1/serverless/create.go`). There is no API for the flag; set it in the
database (the other columns default to false):

```sh
psql "$DATABASE_URL" -c "INSERT INTO tenant_entitlement (tenant_id, serverless_operator) VALUES ('$TENANT_ID', true) ON CONFLICT (tenant_id) DO UPDATE SET serverless_operator = true, updated_at = now()"
```

### 2. Run the Worker locally

```sh
printf 'HATCHET_SIGNING_SECRET=%s\n' "$(openssl rand -hex 32)" > .dev.vars
export SIGNING_SECRET=$(sed -n 's/^HATCHET_SIGNING_SECRET=//p' .dev.vars)
pnpm run dev                                      # wrangler dev, http://localhost:8787
```

Check the signature path before involving the operator. The package computes
`hex(hmac_sha256(secret, raw_body))` over the exact bytes, and the body's
`timestampUnixSeconds` must be within five minutes of the Worker's clock:

```sh
BODY="{\"endpointId\":\"test\",\"timestampUnixSeconds\":\"$(date +%s)\"}"
SIG=$(printf '%s' "$BODY" | openssl dgst -sha256 -hmac "$SIGNING_SECRET" | awk '{print $2}')
curl -s -X POST http://localhost:8787/hatchet/healthcheck -H "X-Hatchet-Signature: $SIG" -d "$BODY" | jq .
```

Expected: `workflows` with `echo` and `sleep-then-echo`, `actions` equal to
`["echo:echo", "sleep-then-echo:sleep-then-echo"]`, `durable.supported: true` and
`runtime.name: "cloudflare-workers"`. An unsigned POST answers `401 {"error": "bad signature"}`.

### 3. Give the operator a URL it may dial

The operator dials endpoints under an SSRF policy (`pkg/operator/safeclient`): `https` only,
port 443 only, and the loopback and private ranges are blocked (`127.0.0.0/8` is in
`DefaultBlockedCIDRs`), so it cannot reach `http://localhost:8787` as it is.
`SERVER_SERVERLESS_OPERATOR_INSECURE_DESTINATIONS=true` switches that dial-time policy off
(plain `http`, any port, loopback), but the REST API checks `healthcheckUrl` and `triggerUrl`
against the same https-on-443 rule when an endpoint is created or updated
(`safeclient.ValidateEndpoint`, which has no insecure mode), so an `http://localhost:8787`
endpoint cannot be registered through the API either way.

Put a public https tunnel in front of the local Worker instead. A public hostname on 443 passes
both checks, so no engine flag is needed:

```sh
cloudflared tunnel --url http://localhost:8787    # prints a https://<random>.trycloudflare.com URL
export WORKER_URL=https://<random>.trycloudflare.com
curl -s -o /dev/null -w '%{http_code}\n' $WORKER_URL/     # 404: the Worker answers through the tunnel
```

The tunnel carries the durable websocket as well.

### 4. Register the endpoint

`POST /api/v1/stable/tenants/{tenant}/serverless/endpoints`
(`api-contracts/openapi/paths/v1/serverless/endpoints.yaml`). `inlineWaitBudgetMs: 500` is
shorter than the durable task's 3 s sleep on purpose, so the eviction path is exercised;
`pollIntervalSeconds: 5` makes the operator notice the Worker quickly.

```sh
curl -s -X POST $API/api/v1/stable/tenants/$TENANT_ID/serverless/endpoints \
  -H "Authorization: Bearer $HATCHET_CLIENT_TOKEN" -H 'Content-Type: application/json' \
  -d "{
    \"name\": \"cloudflare-example\",
    \"kind\": \"CLOUDFLARE_WORKERS\",
    \"healthcheckUrl\": \"$WORKER_URL/hatchet/healthcheck\",
    \"triggerUrl\": \"$WORKER_URL/hatchet/trigger\",
    \"signingSecret\": \"$SIGNING_SECRET\",
    \"pollIntervalSeconds\": 5,
    \"inlineWaitBudgetMs\": 500
  }" | tee /tmp/endpoint.json | jq '{id: .metadata.id, name, enabled}'
export ENDPOINT_ID=$(jq -r .metadata.id /tmp/endpoint.json)
```

Within about 15 seconds the operator has polled the healthcheck and registered the workflows:

```sh
curl -s $API/api/v1/stable/serverless/endpoints/$ENDPOINT_ID \
  -H "Authorization: Bearer $HATCHET_CLIENT_TOKEN" | jq .status
# {"healthy": true, "registeredActions": ["echo:echo", "sleep-then-echo:sleep-then-echo"], ...}
```

`status.error` names the problem when `healthy` is false (a 401 means the endpoint's
`signingSecret` and the Worker's `HATCHET_SIGNING_SECRET` differ).

### 5. Trigger runs

Over REST (`POST /api/v1/stable/tenants/{tenant}/workflow-runs/trigger`):

```sh
curl -s -X POST $API/api/v1/stable/tenants/$TENANT_ID/workflow-runs/trigger \
  -H "Authorization: Bearer $HATCHET_CLIENT_TOKEN" -H 'Content-Type: application/json' \
  -d '{"workflowName": "echo", "input": {"message": "hello"}}' | tee /tmp/run.json | jq -r .run.metadata.id
export RUN_ID=$(jq -r .run.metadata.id /tmp/run.json)
sleep 3
curl -s $API/api/v1/stable/workflow-runs/$RUN_ID \
  -H "Authorization: Bearer $HATCHET_CLIENT_TOKEN" | jq '{status: .run.status, output: .run.output}'
```

Expected: `status: "COMPLETED"` and `output: {"echo": "hello", "workflowRunId": "...", "retryCount": 0}`.
Trigger `sleep-then-echo` the same way and read it back after about 8 seconds: the output is
`{"echo": "hello", "startedAt": "...", "finishedAt": "...", "invocation": 2}`, with `startedAt`
recorded by the first invocation (before the eviction) and `finishedAt` by the second.

Or from a Node script with the TypeScript SDK. The local engine serves gRPC without TLS, so the
client needs `tls_strategy: 'none'` (the SDK defaults to `tls`):

```ts
import { HatchetClient } from '@hatchet-dev/typescript-sdk';

const hatchet = HatchetClient.init({
  token: process.env.HATCHET_CLIENT_TOKEN,
  host_port: 'localhost:7070',
  api_url: 'http://localhost:8888',
  tls_config: { tls_strategy: 'none' },
});

console.log(await hatchet.run('echo', { message: 'hello' }));
```

The same four settings can come from the environment instead: `HATCHET_CLIENT_TOKEN`,
`HATCHET_CLIENT_HOST_PORT`, `HATCHET_CLIENT_API_URL` and `HATCHET_CLIENT_TLS_STRATEGY=none`
(`sdks/typescript/src/util/config-loader/config-loader.ts`).

### 6. Clean up

```sh
curl -s -X DELETE $API/api/v1/stable/serverless/endpoints/$ENDPOINT_ID \
  -H "Authorization: Bearer $HATCHET_CLIENT_TOKEN" | jq .name
```

Stop `cloudflared` and `wrangler dev`; `.dev.vars` and `.wrangler/` are gitignored.

## Deploy to Cloudflare

Prerequisites: Node 22, pnpm, a Cloudflare account on the Workers Paid plan (needed for `cpu_ms`
above 30 s; remove the `[limits]` block to try it on the Free plan, where the 10 ms CPU limit is
enough for `echo` but may not be for anything real), and `wrangler login`.

```sh
openssl rand -hex 32                               # keep the value: it is the endpoint's signingSecret
pnpm wrangler secret put HATCHET_SIGNING_SECRET    # paste it at the prompt
pnpm run deploy                                    # prints https://<name>.<account>.workers.dev
```

Then register the endpoint as in section 4 with the `workers.dev` URL, against whichever Hatchet
instance the token belongs to, and trigger runs as in section 5. Right after the first deploy the
`workers.dev` route can take a minute to go live; until then every request gets an edge 404
whose body is `error code: 1042`.

Optional: `pnpm wrangler secret put HATCHET_ENDPOINT_ID` with the endpoint id makes the Worker
refuse requests and durable upgrades carrying another endpoint id.

## What the Worker verifies

The package checks every request from the operator: the HMAC over the body or the upgrade
headers, the signed timestamp against a five minute window, the endpoint id when
`HATCHET_ENDPOINT_ID` is set, the upgrade nonce against a bounded per-isolate set, and the
first durable frame against the signed upgrade. The package README's "What the package
verifies" section has the details and the production advice for nonces.

## Watch it run

```sh
pnpm run tail        # deployed Worker; wrangler dev prints the same lines in its own terminal
```

shows each request. The adapter logs a task's `ctx.logger` calls, failed tasks and durable
evictions; healthchecks and successful triggers are silent. For the durable task expect, on the
first invocation, `[hatchet] durable task <id>: inline wait budget of 500ms elapsed, evicting`,
and nothing on the second, which completes.

## Tests without Hatchet

`@hatchet-dev/serverless/testing` invokes the tasks the way the operator would, with no engine
and no tunnel; see the package README.
