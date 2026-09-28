# Hatchet serverless endpoint in Next.js on Vercel

A Next.js 15 App Router app that serves Hatchet tasks without running a worker process, built
on `@hatchet-dev/serverless`, and triggers them from a server action with the SDK's core
client. The Hatchet serverless operator polls the endpoint for the workflows it serves and
delivers assigned tasks to it over signed HTTPS requests.

- `echo`: a non-durable task. The operator POSTs the task, the route returns the message with
  the run id and retry count.
- `sleep-then-echo`: a durable task. It records a timestamp, sleeps 3 seconds, spawns `echo` as
  a child run and waits for its output, then returns everything. The sleep exceeds the
  endpoint's inline wait budget, so the first invocation evicts itself and the engine re-invokes
  the task when the sleep is over. It runs on a websocket the operator dials, which on Vercel
  needs Fluid compute (see "Durable tasks on Vercel" below).

The tasks are the same as in `examples/serverless/cloudflare-workers`, with the same names and
outputs, so one endpoint registration works for either runtime.

## Layout

| File                                     | Purpose                                                                                                                                                                                                                              |
| ---------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `hatchet/tasks.ts`                       | The declarations, written with `hatchet.task` and `hatchet.durableTask` from `@hatchet-dev/serverless`. The same file works on a regular worker.                                                                                    |
| `app/api/hatchet/[...hatchet]/route.ts`  | `export const { GET, POST } = vercel({ workflows })`: `POST` serves the healthcheck and non-durable triggers, `GET` the durable websocket upgrade; the secret comes from `HATCHET_SIGNING_SECRET`. `maxDuration` caps one invocation. |
| `hatchet/client.ts`                      | The SDK's core client (`@hatchet-dev/typescript-sdk/core`, Connect over fetch, no gRPC) built from `HATCHET_CLIENT_TOKEN` and friends. Only the triggering side has the token.                                                       |
| `app/actions.ts`, `app/trigger-form.tsx` | A server action that triggers `echo` or `sleep-then-echo` and waits for the output, and the form on the page that calls it.                                                                                                        |
| `app/api/echo/route.ts`                  | `GET /api/echo?message=hello`: the same trigger for curl.                                                                                                                                                                            |
| `next.config.ts`                         | Keeps the package, the SDK, `ws` and `@vercel/functions` out of the server bundle (`serverExternalPackages`), since the adapter loads the last two at runtime.                                                                          |

The wire contract is `api-contracts/v1/serverless.proto` plus
`pkg/serverlessoperator/contract/http.go` (headers, the signature scheme) and
`pkg/serverlessoperator/durable/protocol.go` (close codes); the package generates its types
from the proto.

## Build the package first

`@hatchet-dev/serverless` is not published yet; this example links it from the repository, and
both it and the example take the TypeScript SDK from its build output. From the repository root:

```sh
cd sdks/typescript && pnpm install && pnpm tsc:build && pnpm prepublish
cd ../typescript-serverless && pnpm install && pnpm build
cd ../../examples/serverless/nextjs && pnpm install
pnpm run typecheck && pnpm run build
```

## Secrets: what lives where

The endpoint holds one secret, `HATCHET_SIGNING_SECRET`, the HMAC key the operator signs every
request with. The trigger side (the server action, the `/api/echo` route) holds the tenant API
token, `HATCHET_CLIENT_TOKEN`. The two never meet: the endpoint has no Hatchet client, and the
operator never sends a token to it. In this app both live in the same deployment because it
both serves and triggers; a project that only serves needs the signing secret alone.

- `next dev` and `vc dev` read both from `.env.local` (gitignored).
- On Vercel they are project environment variables.

The signing secret is also the endpoint's `signingSecret` when it is registered in Hatchet. It
must be 32 characters or longer; `openssl rand -hex 32` makes a fitting one.

## Run against a local engine

### What you need

- A Hatchet engine and API built from this repository with the in-engine serverless operator
  on: `SERVER_SERVERLESS_OPERATOR_ENABLED=true` and
  `SERVER_SERVERLESS_OPERATOR_ALLOW_EMPTY_INFRA_CIDRS=true` (both documented in
  `frontend/docs/content/docs/self-hosting/configuration-options.mdx`). The commands below
  assume the REST API at `http://localhost:8888` and the engine's gRPC/Connect port at
  `localhost:7070` (the defaults), served without TLS (`SERVER_GRPC_INSECURE=true`).
- An API token for the tenant (dashboard: Settings, API Tokens) and the tenant id.
- `psql` access to the engine database, `jq`, `curl`, `openssl`, and `cloudflared` for
  section 3.

```sh
export API=http://localhost:8888
export TENANT_ID=<tenant id>
export HATCHET_CLIENT_TOKEN=<api token>
export DATABASE_URL=<the engine's postgres URL>
curl -sf -H "Authorization: Bearer $HATCHET_CLIENT_TOKEN" $API/api/v1/tenants/$TENANT_ID >/dev/null && echo token ok
```

### 1. Entitle the tenant

Endpoint creation answers `403 the serverless operator is not enabled for this tenant` until
`tenant_entitlement.serverless_operator` is true for the tenant. There is no API for the flag;
set it in the database:

```sh
psql "$DATABASE_URL" -c "INSERT INTO tenant_entitlement (tenant_id, serverless_operator) VALUES ('$TENANT_ID', true) ON CONFLICT (tenant_id) DO UPDATE SET serverless_operator = true, updated_at = now()"
```

### 2. Run the app locally

The core client is configured from the environment by `hatchet/client.ts`. The local engine
serves Connect on its gRPC port without TLS, so it needs the engine URL and
`HATCHET_CLIENT_TLS_STRATEGY=none` (the client defaults to `https` at the address in the
token's `grpc_broadcast_address` claim):

```sh
cat > .env.local <<EOF
HATCHET_SIGNING_SECRET=$(openssl rand -hex 32)
HATCHET_CLIENT_TOKEN=$HATCHET_CLIENT_TOKEN
HATCHET_CLIENT_SERVER_URL=http://localhost:7070
HATCHET_CLIENT_TLS_STRATEGY=none
EOF
export SIGNING_SECRET=$(sed -n 's/^HATCHET_SIGNING_SECRET=//p' .env.local)
pnpm run dev                                      # next dev, http://localhost:3000
```

`next dev` serves the healthcheck and non-durable triggers. The durable websocket upgrade is
only available under `vc dev` (see "Durable tasks on Vercel"); outside Vercel the adapter
reports `durable.supported: false`, so the operator delivers `echo` and leaves
`sleep-then-echo` alone.

Check the signature path before involving the operator. The package computes
`hex(hmac_sha256(secret, raw_body))` over the exact bytes, and the body's
`timestampUnixSeconds` must be within five minutes of the app's clock:

```sh
BODY="{\"endpointId\":\"test\",\"timestampUnixSeconds\":\"$(date +%s)\"}"
SIG=$(printf '%s' "$BODY" | openssl dgst -sha256 -hmac "$SIGNING_SECRET" | awk '{print $2}')
curl -s -X POST http://localhost:3000/api/hatchet/healthcheck -H "X-Hatchet-Signature: $SIG" -d "$BODY" | jq .
```

Expected: `workflows` with `echo` and `sleep-then-echo`, `actions` equal to
`["echo:echo", "sleep-then-echo:sleep-then-echo"]`, `runtime.name: "vercel"` and, under
`next dev`, `durable: {}` (protojson leaves a false `supported` out). An unsigned POST answers
`401 {"error": "bad signature"}`.

### 3. Give the operator a URL it may dial

The operator dials endpoints under an SSRF policy (`pkg/operator/safeclient`): `https` only,
port 443 only, loopback and private ranges blocked, and the REST API validates
`healthcheckUrl` and `triggerUrl` against the same rule, so `http://localhost:3000` can neither
be registered nor reached. Put a public https tunnel in front of the local app:

```sh
cloudflared tunnel --url http://localhost:3000    # prints a https://<random>.trycloudflare.com URL
export APP_URL=https://<random>.trycloudflare.com
curl -s -o /dev/null -w '%{http_code}\n' $APP_URL/     # 200: the page answers through the tunnel
```

The tunnel carries the durable websocket as well (under `vc dev`).

### 4. Register the endpoint

`POST /api/v1/stable/tenants/{tenant}/serverless/endpoints`. `inlineWaitBudgetMs: 500` is
shorter than the durable task's 3 s sleep on purpose, so the eviction path is exercised;
`pollIntervalSeconds: 5` makes the operator notice the app quickly. The `kind` is `GENERIC_HTTP`:
the API has no Vercel-specific kind, and the adapter applies the Vercel constraints itself.

```sh
curl -s -X POST $API/api/v1/stable/tenants/$TENANT_ID/serverless/endpoints \
  -H "Authorization: Bearer $HATCHET_CLIENT_TOKEN" -H 'Content-Type: application/json' \
  -d "{
    \"name\": \"nextjs-example\",
    \"kind\": \"GENERIC_HTTP\",
    \"healthcheckUrl\": \"$APP_URL/api/hatchet/healthcheck\",
    \"triggerUrl\": \"$APP_URL/api/hatchet/trigger\",
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
`signingSecret` and the app's `HATCHET_SIGNING_SECRET` differ).

### 5. Trigger runs

From the app itself: open `http://localhost:3000`, pick `echo`, type a message and press Run.
The server action triggers the run through the core client and shows the output after the
operator delivered the task to this same app through the tunnel. Or with curl:

```sh
curl -s "http://localhost:3000/api/echo?message=hello" | jq .
# {"runId": "...", "output": {"echo": "hello", "workflowRunId": "...", "retryCount": 0}}
```

Or over REST, the way the Cloudflare example does it:

```sh
curl -s -X POST $API/api/v1/stable/tenants/$TENANT_ID/workflow-runs/trigger \
  -H "Authorization: Bearer $HATCHET_CLIENT_TOKEN" -H 'Content-Type: application/json' \
  -d '{"workflowName": "echo", "input": {"message": "hello"}}' | jq -r .run.metadata.id
```

`sleep-then-echo` completes only when the endpoint serves durable tasks, which is under
`vc dev` or on Vercel with Fluid compute. Its output is
`{"echo": "hello", "child": {...}, "startedAt": "...", "finishedAt": "...", "invocation": 2}`
(or 3 when the child wait evicted too), as described in the Cloudflare example's README.

### 6. Clean up

```sh
curl -s -X DELETE $API/api/v1/stable/serverless/endpoints/$ENDPOINT_ID \
  -H "Authorization: Bearer $HATCHET_CLIENT_TOKEN" | jq .name
```

Stop `cloudflared` and the dev server; `.env.local` and `.next/` are gitignored.

## Durable tasks on Vercel

Vercel Functions accept websocket upgrades on Fluid compute (public beta; the default for
projects created after April 2025). Next.js has no upgrade API of its own, so the adapter uses
`experimental_upgradeWebSocket` from `@vercel/functions`, which needs the `ws` package. Both are
in this example's `dependencies`.

- **`vc dev` vs `next dev`**: `experimental_upgradeWebSocket` runs locally only under the Vercel
  CLI (`vc dev`, CLI 54.14.2 or later; `pnpm run dev:vercel` here). Under `next dev` the upgrade
  never reaches the route, so the adapter advertises `durable.supported: false` whenever the
  `VERCEL` environment variable is absent, and refuses an upgrade with
  `426 {"error": "websocket upgrade unavailable: ...", "retry": false}` plus a logged hint if
  one arrives anyway. Pass `durable: true` to `vercel()` to force advertising.
- **Fluid compute**: on a project without it the upgrade fails at request time with the same
  426. The adapter cannot detect this from a healthcheck, so pass `durable: false` on such a
  project and let another endpoint (a Cloudflare Worker, a Node server) serve the durable tasks.
- **`maxDuration`**: a durable invocation lives on its websocket until it finishes or evicts
  itself, so `maxDuration` in `route.ts` is the ceiling on one invocation: 300 s on Hobby, up to
  800 s on Pro and Enterprise. The endpoint's `inlineWaitBudgetMs` must stay well below it; the
  invocation evicts itself when a wait exceeds the budget and the engine re-invokes the task
  later, so long sleeps never need a long function.
- **Body size**: request and response bodies through Vercel Functions are capped at 4.5 MB;
  the relay's frames are capped at 4 MiB by the operator, which the adapter passes to
  `maxPayload`.
- **Lifetime**: the adapter awaits the relay inside the upgrade callback, so the route's
  response completes only when the invocation does, and registers the same promise with
  `waitUntil` from `@vercel/functions`. `after()` from `next/server` is not used: it schedules
  work for after the response, while the relay is already running when the 101 goes out, and
  the adapter also serves non-Next.js functions.

## Deploy to Vercel

Prerequisites: Node 22, pnpm, a Vercel account with Fluid compute on for the project, and
`vercel login`. The package is linked from the repository here, so deploying this exact
directory needs the built `sdks/typescript-serverless/dist` and `sdks/typescript/dist` to be
part of the upload; a real project depends on the published package instead.

```sh
openssl rand -hex 32                               # keep the value: it is the endpoint's signingSecret
vercel env add HATCHET_SIGNING_SECRET production   # paste it at the prompt
vercel env add HATCHET_CLIENT_TOKEN production     # the tenant API token, for the trigger side
vercel deploy --prod                               # prints https://<project>.vercel.app
```

Then register the endpoint as in section 4 with the `vercel.app` URL (`kind: "GENERIC_HTTP"`),
against whichever Hatchet instance the token belongs to, and trigger runs as in section 5.

Optional: `HATCHET_ENDPOINT_ID` with the endpoint id makes the app refuse requests and durable
upgrades carrying another endpoint id.

## Outside Vercel

The same app on your own Node server (`next start` behind a proxy, or a custom server) cannot
take the websocket through a route handler. Use `@hatchet-dev/serverless/node` there:
`createServer({ workflows })` on its own port, or `nodeHandler({ workflows })` as middleware on
a custom server whose `upgrade` event is wired to `handler.upgrade`. The package README's
"Adapters" section has the table.

## Tests without Hatchet

`@hatchet-dev/serverless/testing` invokes the tasks the way the operator would, with no engine
and no tunnel; see the package README.
