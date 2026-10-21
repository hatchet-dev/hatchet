/**
 * The Hatchet serverless endpoint. The operator (pkg/serverlessoperator) is the only caller:
 *
 *   POST /api/hatchet/healthcheck   signed; answers the workflows this endpoint serves
 *   POST /api/hatchet/trigger       signed; runs one non-durable task and returns its output
 *   GET  /api/hatchet/trigger       signed websocket upgrade; one durable or `streams` invocation
 *
 * The signing secret comes from HATCHET_SIGNING_SECRET. The tasks' Hatchet client (what
 * `ctx.runChild` in parent-echo triggers through) is the same core client the app triggers
 * with, built from HATCHET_CLIENT_TOKEN. The route is read from the catch-all segment, so this
 * file can move without changing anything else.
 */
import { vercel } from "@hatchet-dev/serverless/vercel";
import { hatchetClient } from "@/hatchet/client";
import { parentEcho, workflows } from "@/hatchet/tasks";

export const { GET, POST } = vercel({
  workflows,
  // parent-echo awaits a child run, so the operator must invoke it over the websocket.
  streams: [parentEcho],
  client: () => (process.env.HATCHET_CLIENT_TOKEN ? hatchetClient() : undefined),
});

// A durable invocation lives on its websocket until it finishes or evicts itself, so this is
// the ceiling on one invocation. 300 s is the Hobby maximum; Pro and Enterprise allow 800.
export const maxDuration = 300;
