/**
 * Example Hatchet serverless endpoint on Cloudflare Workers.
 *
 * Routes (served by @hatchet-dev/serverless/cloudflare):
 *   POST /hatchet/healthcheck   signed; answers the workflows this endpoint serves
 *   POST /hatchet/trigger       signed; runs one pure task (`hatchetClient: false`) and returns its output
 *   GET  /hatchet/trigger       signed websocket upgrade; one invocation of any other task, durable or not
 *   anything else               404
 *
 * The operator (pkg/serverlessoperator) is the only caller. It polls the healthcheck, dials the
 * websocket for every task that talks to Hatchet (the default, durable or not) and POSTs the
 * tasks declared with `hatchetClient: false` to the trigger URL. The Worker holds two secrets:
 * HATCHET_SIGNING_SECRET, the HMAC key the operator signs every request with, and
 * HATCHET_CLIENT_TOKEN, the tenant API token the adapter builds the tasks' Hatchet client from
 * (`ctx.runChild`, `ctx.putStream`, `ctx.log`). The operator never sends a token to the Worker.
 * Under `wrangler dev`, HATCHET_CLIENT_HOST_PORT and HATCHET_CLIENT_TLS_STRATEGY=none in
 * .dev.vars point that client at a local engine served over plain HTTP.
 */
import { cloudflare } from "@hatchet-dev/serverless/cloudflare";
import { workflows } from "./tasks";

export default cloudflare({ workflows });
