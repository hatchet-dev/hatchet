/**
 * The Hatchet client the app triggers runs with. It is the SDK's core client (unary Connect
 * calls over fetch, no gRPC), so it runs inside a Vercel Function, and it is built from the
 * environment here because the core client reads nothing itself:
 *
 *   HATCHET_CLIENT_TOKEN         the tenant API token (required)
 *   HATCHET_CLIENT_SERVER_URL    the engine's base URL; defaults to the token's
 *                                grpc_broadcast_address claim over https
 *   HATCHET_CLIENT_TLS_STRATEGY  "none" for a local engine served over plain HTTP
 *
 * The endpoint under app/api/hatchet hands the same client to its tasks, so parent-echo can
 * trigger its child run; the operator itself never sends a token to the endpoint.
 */
import { HatchetCore } from "@hatchet-dev/typescript-sdk/core";

let client: HatchetCore | undefined;

export function hatchetClient(): HatchetCore {
  if (!client) {
    const token = process.env.HATCHET_CLIENT_TOKEN;

    if (!token) {
      throw new Error("HATCHET_CLIENT_TOKEN is not set");
    }

    client = new HatchetCore({
      token,
      serverUrl: process.env.HATCHET_CLIENT_SERVER_URL,
      tls: process.env.HATCHET_CLIENT_TLS_STRATEGY === "none" ? { strategy: "none" } : undefined,
    });
  }

  return client;
}
