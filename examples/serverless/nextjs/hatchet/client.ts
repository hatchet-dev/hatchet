/**
 * The Hatchet client the app triggers runs with. It is the SDK's core client (unary Connect
 * calls over fetch, no gRPC), so it runs inside a Vercel Function. `clientConfigFromEnv` reads
 * the same variables the endpoint's adapter defaults to:
 *
 *   HATCHET_CLIENT_TOKEN         the tenant API token (required)
 *   HATCHET_CLIENT_HOST_PORT     the engine's host:port when it is not the one the token names
 *   HATCHET_CLIENT_TLS_STRATEGY  "none" for a local engine served over plain HTTP
 *
 * The endpoint under app/api/hatchet builds its tasks' client from them as well, so parent-echo
 * can trigger its child run; the operator itself never sends a token to the endpoint.
 */
import { HatchetCore, clientConfigFromEnv } from "@hatchet-dev/serverless";

let client: HatchetCore | undefined;

export function hatchetClient(): HatchetCore {
  if (!client) {
    const config = clientConfigFromEnv(process.env);

    if (!config) {
      throw new Error("HATCHET_CLIENT_TOKEN is not set");
    }

    client = new HatchetCore(config);
  }

  return client;
}
