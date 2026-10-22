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
 */
import { cloudflare } from "@hatchet-dev/serverless/cloudflare";
import { workflows } from "./tasks";

/** The Worker's bindings: the two secrets, and the local-engine address for `wrangler dev`. */
interface Env {
  HATCHET_SIGNING_SECRET?: string;
  HATCHET_CLIENT_TOKEN?: string;
  /**
   * The engine's base URL when it is not the one the token names. A local engine is served
   * over plain HTTP, which the client refuses unless `tls` is `{ strategy: "none" }`.
   */
  HATCHET_CLIENT_SERVER_URL?: string;
}

export default cloudflare<Env>({
  workflows,
  client: (env) => {
    if (!env.HATCHET_CLIENT_TOKEN) {
      return undefined;
    }

    const serverUrl = env.HATCHET_CLIENT_SERVER_URL;

    if (!serverUrl) {
      return { token: env.HATCHET_CLIENT_TOKEN };
    }

    return {
      token: env.HATCHET_CLIENT_TOKEN,
      serverUrl,
      tls: { strategy: serverUrl.startsWith("http:") ? ("none" as const) : ("tls" as const) },
    };
  },
});
