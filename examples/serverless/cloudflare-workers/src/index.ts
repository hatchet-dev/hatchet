/**
 * Example Hatchet serverless endpoint on Cloudflare Workers.
 *
 * Routes:
 *   POST /hatchet/healthcheck   signed; answers the workflows this endpoint serves
 *   POST /hatchet/trigger       signed; runs one non-durable task and returns its output
 *   GET  /hatchet/trigger       websocket upgrade (signed headers); runs one durable task
 *   anything else               404
 *
 * The operator (pkg/serverlessoperator) is the only caller. It polls the healthcheck, and for
 * every assigned task either POSTs the trigger URL (non-durable) or dials it as a websocket
 * (durable). See src/hatchet.ts for the wire contract.
 */

import {
  type AssignedAction,
  type DoneOutcome,
  DurableClient,
  Evicted,
  type HealthcheckRequest,
  type OutboundFrame,
  SIGNATURE_HEADER,
  type TriggerRequest,
  type WorkflowDefinition,
  healthcheckResponse,
  json,
  parseActionInput,
  stripNamespace,
  triggerError,
  verifySignedBody,
  verifyUpgradeSignature,
} from "./hatchet";

export interface Env {
  /** `wrangler secret put HATCHET_SIGNING_SECRET`; the endpoint's signingSecret in Hatchet. */
  HATCHET_SIGNING_SECRET: string;
  /** Optional: when set, requests and upgrades carrying any other endpoint id are refused with 403. */
  HATCHET_ENDPOINT_ID?: string;
}

// ---------------------------------------------------------------------------------------
// Workflows this endpoint serves. Names and actions are un-prefixed; the operator applies
// the endpoint's namespace (`<uuid>_echo`, `<uuid>_echo:echo`) when it registers them.
// ---------------------------------------------------------------------------------------

const WORKFLOWS: WorkflowDefinition[] = [
  {
    name: "echo",
    description: "Non-durable: returns its input",
    tasks: [{ readableId: "echo", action: "echo:echo", timeout: "60s" }],
  },
  {
    name: "sleep-then-echo",
    description: "Durable: memoizes a timestamp, sleeps 3 seconds, returns both",
    tasks: [{ readableId: "run", action: "sleep-then-echo:run", timeout: "5m", isDurable: true }],
  },
];

interface EchoInput {
  input?: unknown;
}

/** Non-durable tasks, keyed by un-prefixed action id. */
const TASKS: Record<string, (action: AssignedAction, input: EchoInput) => Promise<unknown>> = {
  "echo:echo": async (action, input) => ({
    echo: input.input ?? null,
    workflowRunId: action.workflowRunId ?? null,
    retryCount: action.retryCount ?? 0,
  }),
};

/** Durable tasks, keyed by un-prefixed action id. */
const DURABLE_TASKS: Record<string, (ctx: DurableClient, input: EchoInput) => Promise<unknown>> = {
  "sleep-then-echo:run": async (ctx, input) => {
    // Invocation 1 computes this; invocation 2 (after the eviction) gets it from the log.
    const startedAt = await ctx.memo("started-at", () => new Date().toISOString());

    // Invocation 1: wait_for, ack, then the inline budget elapses, evict, done evicted.
    // Invocation 2: wait_for, ack and entry_completed arrive together; continues at once.
    await ctx.sleep(3000);

    return {
      echo: input.input ?? null,
      startedAt,
      finishedAt: new Date().toISOString(),
      invocation: ctx.invocation,
    };
  },
};

// ---------------------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------------------

export default {
  async fetch(request: Request, env: Env, ctx: ExecutionContext): Promise<Response> {
    const url = new URL(request.url);

    if (!env.HATCHET_SIGNING_SECRET) {
      return json({ error: "HATCHET_SIGNING_SECRET is not set on the Worker" }, 500);
    }

    if (url.pathname === "/hatchet/healthcheck") {
      return handleHealthcheck(request, env);
    }

    if (url.pathname === "/hatchet/trigger") {
      // Step 1 (durable): the operator dials the trigger URL with Upgrade: websocket.
      if (request.headers.get("Upgrade")?.toLowerCase() === "websocket") {
        return handleDurableUpgrade(request, env, ctx);
      }

      return handleTrigger(request, env);
    }

    return new Response("not found", { status: 404 });
  },
};

/**
 * POST healthcheck_url. Body {"endpointId", "namespace", "timestamp"} signed with
 * X-Hatchet-Signature; the timestamp must be within five minutes of now. The response lists
 * the workflows; the operator registers them on change.
 */
async function handleHealthcheck(request: Request, env: Env): Promise<Response> {
  if (request.method !== "POST") {
    return new Response("method not allowed", { status: 405 });
  }

  // Read the raw body first: the signature covers the exact bytes.
  const body = await request.text();

  const verified = await verifySignedBody(body, request.headers.get(SIGNATURE_HEADER), env.HATCHET_SIGNING_SECRET, {
    endpointId: env.HATCHET_ENDPOINT_ID,
  });

  if (!verified.ok) {
    return json({ error: verified.reason }, verified.status);
  }

  const req = JSON.parse(body) as HealthcheckRequest;

  console.log(`healthcheck endpoint=${req.endpointId} namespace=${req.namespace}`);

  return json(healthcheckResponse(WORKFLOWS));
}

/**
 * POST trigger_url for a non-durable task. The envelope carries the protojson AssignedAction
 * with a namespaced action id and a timestamp that must be within five minutes of now;
 * strip the namespace, run the task, and answer:
 *   200 + JSON body   COMPLETED with that output
 *   4xx               FAILED, not retried (except 408, 425, 429)
 *   5xx               FAILED, retried
 * {"error", "retry"} in a non-2xx body overrides the default message and retry decision.
 *
 * Within the window a delivery can be repeated; a task with side effects should key them on
 * (endpointId, taskRunExternalId, retryCount).
 */
async function handleTrigger(request: Request, env: Env): Promise<Response> {
  if (request.method !== "POST") {
    return new Response("method not allowed", { status: 405 });
  }

  const body = await request.text();

  const verified = await verifySignedBody(body, request.headers.get(SIGNATURE_HEADER), env.HATCHET_SIGNING_SECRET, {
    endpointId: env.HATCHET_ENDPOINT_ID,
  });

  if (!verified.ok) {
    return triggerError(verified.status, verified.reason, false);
  }

  const envelope = JSON.parse(body) as TriggerRequest;
  const action = envelope.action;
  const actionId = stripNamespace(action.actionId, envelope.namespace);

  console.log(`trigger action=${actionId} task=${action.taskRunExternalId} retry=${action.retryCount ?? 0}`);

  if (action.actionType && action.actionType !== "START_STEP_RUN") {
    return triggerError(400, `unsupported action type ${action.actionType}`, false);
  }

  const task = TASKS[actionId];

  if (!task) {
    return triggerError(404, `no task registered for action ${actionId}`, false);
  }

  try {
    const output = await task(action, parseActionInput<EchoInput>(action).input as EchoInput);

    return json(output ?? {});
  } catch (err) {
    return triggerError(500, errorMessage(err), true);
  }
}

/**
 * GET trigger_url with Upgrade: websocket, for a durable task.
 *
 * Step 2: verify the signed headers (timestamp within five minutes, nonce not seen before,
 *         task id, invocation) before accepting.
 * Step 3: accept with WebSocketPair and answer 101; the socket lives for the whole invocation
 *         and there is no wall-clock limit while the operator stays connected.
 * Step 4: the first frame carries the action; check it names the verified task and
 *         invocation, then start the task.
 * Step 5: relay memo / wait_for / evict_invocation frames through DurableClient.
 * Step 6: finish with exactly one done frame; the operator closes the socket (1000).
 */
async function handleDurableUpgrade(request: Request, env: Env, ctx: ExecutionContext): Promise<Response> {
  const verified = await verifyUpgradeSignature(request.headers, env.HATCHET_SIGNING_SECRET, {
    endpointId: env.HATCHET_ENDPOINT_ID,
  });

  if (!verified.ok) {
    console.log(`upgrade refused: ${verified.reason}`);
    return new Response(verified.reason, { status: verified.status });
  }

  console.log(`upgrade accepted task=${verified.taskId} invocation=${verified.invocation}`);

  const pair = new WebSocketPair();
  const [client, server] = Object.values(pair) as [WebSocket, WebSocket];

  server.accept();

  let durable: DurableClient | null = null;

  server.addEventListener("message", (event: MessageEvent) => {
    if (typeof event.data !== "string") {
      server.close(1003, "text frames only");
      return;
    }

    if (durable) {
      durable.onMessage(event.data);
      return;
    }

    // The first frame: {"first": {"action", "namespace", "invocationCount", "inlineWaitBudgetMs"}}.
    let first: OutboundFrame["first"];

    try {
      first = (JSON.parse(event.data) as OutboundFrame).first;
    } catch {
      server.close(1003, "malformed first frame");
      return;
    }

    if (!first) {
      server.close(1003, "expected a first frame");
      return;
    }

    const taskId = first.action.taskRunExternalId;
    const client = new DurableClient(server, first, (msg) => console.log(`durable ${taskId}: ${msg}`));

    // The upgrade signature covered the task id and invocation in the headers, not this
    // frame: only a frame that names them may run.
    const mismatch = client.assertMatches(verified);

    if (mismatch) {
      console.log(`upgrade refused after the first frame: ${mismatch}`);
      server.close(1008, mismatch);
      return;
    }

    durable = client;

    console.log(
      `durable first frame task=${durable.taskId} invocation=${durable.invocation} budget=${durable.inlineWaitBudgetMs}ms`,
    );

    // The open socket is what keeps the isolate running while the task awaits engine
    // responses; waitUntil additionally registers the task promise with the runtime. It is
    // called after the 101 went out, so tolerate a runtime that refuses it at this point.
    const task = runDurable(durable);

    try {
      ctx.waitUntil(task);
    } catch {
      // The promise still runs; the socket keeps the request context alive.
    }
  });

  server.addEventListener("close", (event: CloseEvent) => {
    console.log(`durable socket closed code=${event.code} reason=${event.reason}`);
    durable?.onClose(`code ${event.code}`);
  });

  server.addEventListener("error", () => {
    durable?.onClose("socket error");
  });

  return new Response(null, { status: 101, webSocket: client });
}

async function runDurable(durable: DurableClient): Promise<void> {
  const actionId = stripNamespace(durable.action.actionId, durable.namespace);
  const task = DURABLE_TASKS[actionId];

  if (!task) {
    durable.done({ error: `no durable task registered for action ${actionId}`, retry: false });
    return;
  }

  try {
    const output = await task(durable, parseActionInput<EchoInput>(durable.action).input as EchoInput);
    const outcome: DoneOutcome = { output: JSON.stringify(output ?? {}) };

    console.log(`durable ${durable.taskId}: done invocation=${durable.invocation}`);
    durable.done(outcome);
  } catch (err) {
    if (err instanceof Evicted) {
      // Endpoint eviction already sent done {"status": "evicted"}; a server eviction is
      // followed by the operator closing 4001. Nothing more to send either way.
      console.log(`durable ${durable.taskId}: ${err.message}`);
      return;
    }

    // Engine errors (non-determinism on replay) and task failures are permanent.
    console.log(`durable ${durable.taskId}: failed: ${errorMessage(err)}`);
    durable.done({ error: errorMessage(err), retry: false });
  }
}

function errorMessage(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}
