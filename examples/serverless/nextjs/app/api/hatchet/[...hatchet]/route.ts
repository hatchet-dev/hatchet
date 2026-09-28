/**
 * The Hatchet serverless endpoint. The operator (pkg/serverlessoperator) is the only caller:
 *
 *   POST /api/hatchet/healthcheck   signed; answers the workflows this endpoint serves
 *   POST /api/hatchet/trigger       signed; runs one non-durable task and returns its output
 *   GET  /api/hatchet/trigger       signed websocket upgrade; runs one durable invocation
 *
 * The signing secret comes from HATCHET_SIGNING_SECRET. The route is read from the catch-all
 * segment, so this file can move without changing anything else.
 */
import { vercel } from "@hatchet-dev/serverless/vercel";
import { workflows } from "@/hatchet/tasks";

export const { GET, POST } = vercel({ workflows });

// A durable invocation lives on its websocket until it finishes or evicts itself, so this is
// the ceiling on one invocation. 300 s is the Hobby maximum; Pro and Enterprise allow 800.
export const maxDuration = 300;
