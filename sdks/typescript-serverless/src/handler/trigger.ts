/**
 * POST trigger_url for a non-durable task: verify the signature over the raw body, decode
 * the envelope, strip the namespace, look the task up, run it under a `Context` and map the
 * outcome onto the status codes pkg/serverlessoperator/delivery.go classifyResponse reads.
 */
import { Context, NonRetryableError } from '@hatchet-dev/typescript-sdk/edge/index.js';
import { ActionType } from '../generated/proto/dispatcher';
import { ServerlessTriggerRequest } from '../generated/proto/v1/serverless';
import {
  SIGNATURE_HEADER,
  TRIGGER_ENVELOPE_VERSION,
  namespacePrefix,
  stripNamespace,
} from './contract';
import { toSdkAction } from './action';
import { ServerlessRuntime, type ConsoleLike } from './context';
import { isServerlessLimitationError } from './errors';
import { errorMessage, json, triggerError } from './http';
import type { Registry } from './registry';
import { verifySignedBody } from './signature';

export interface TriggerOptions {
  registry: Registry;
  secret: string;
  /** When set, envelopes carrying another endpoint id are refused with 403. */
  endpointId?: string;
  console?: ConsoleLike;
}

export async function handleTrigger(request: Request, options: TriggerOptions): Promise<Response> {
  if (request.method !== 'POST') {
    return triggerError(405, 'method not allowed', false);
  }

  // The signature covers the exact bytes, so read the body before parsing it. The body's
  // timestamp must be within the request window and its endpoint id must be this endpoint's.
  const body = await request.text();
  const verified = await verifySignedBody(
    body,
    request.headers.get(SIGNATURE_HEADER),
    options.secret,
    { endpointId: options.endpointId }
  );

  if (!verified.ok) {
    return triggerError(verified.status, verified.reason, false);
  }

  let envelope: ServerlessTriggerRequest;

  try {
    envelope = ServerlessTriggerRequest.fromJSON(verified.json);
  } catch (err) {
    return triggerError(400, `malformed trigger request: ${errorMessage(err)}`, false);
  }

  if (envelope.version !== TRIGGER_ENVELOPE_VERSION) {
    return triggerError(
      400,
      `unsupported trigger envelope version ${envelope.version}; this package speaks version ${TRIGGER_ENVELOPE_VERSION}`,
      false
    );
  }

  if (!envelope.action) {
    return triggerError(400, 'trigger request carries no action', false);
  }

  if (envelope.action.actionType !== ActionType.START_STEP_RUN) {
    return triggerError(
      400,
      `unsupported action type ${ActionType[envelope.action.actionType] ?? envelope.action.actionType}`,
      false
    );
  }

  const { registry } = options;
  const actionId = stripNamespace(envelope.action.actionId, envelope.namespace);

  if (registry.durableActions.has(actionId)) {
    return triggerError(
      422,
      `action ${actionId} is a durable task; durable tasks run over the websocket relay, which this version of @hatchet-dev/serverless does not provide`,
      false
    );
  }

  const runner = registry.runners.get(actionId);

  if (!runner || !registry.served.has(actionId)) {
    return triggerError(404, `no task served for action ${actionId}`, false);
  }

  let ctx: Context<unknown, unknown>;

  try {
    ctx = new Context(
      toSdkAction(envelope.action, envelope.namespace),
      new ServerlessRuntime({
        namespace: namespacePrefix(envelope.namespace),
        hasWorkflow: registry.hasWorkflow,
        console: options.console,
      })
    );
  } catch (err) {
    return triggerError(400, `could not build the task context: ${errorMessage(err)}`, false);
  }

  // The operator cancels a delivery by dropping the request; abort the task with it.
  try {
    request.signal?.addEventListener('abort', () => ctx.abortController.abort());
  } catch {
    // Some runtimes have no request signal; the task then runs to completion.
  }

  try {
    const output = await runner(ctx);

    if (output === undefined) {
      return new Response(null, { status: 204 });
    }

    return json(output);
  } catch (err) {
    if (err instanceof NonRetryableError || isServerlessLimitationError(err)) {
      return triggerError(422, errorMessage(err), false);
    }

    (options.console ?? console).error(
      `[hatchet] task ${actionId} (run ${envelope.action.workflowRunId}) failed: ${errorMessage(err)}`,
      err
    );

    return triggerError(500, errorMessage(err), true);
  }
}
