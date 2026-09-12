/**
 * `@hatchet-dev/serverless/testing`: invokes a handler the way the operator would, without
 * Hatchet. Requests are signed with the endpoint secret, action ids are namespaced, and
 * responses are classified the way pkg/serverlessoperator/delivery.go does.
 * @module Testing
 */
import type {
  BaseWorkflowDeclaration,
  CreateWorkflowTaskOpts,
  TaskWorkflowDeclaration,
  WorkflowDeclaration,
} from '@hatchet-dev/typescript-sdk/edge/index.js';
import { createActionId } from '@hatchet-dev/typescript-sdk/edge/index.js';
import { ActionType, AssignedAction } from '../generated/proto/dispatcher';
import {
  ServerlessHealthcheckRequest,
  ServerlessHealthcheckResponse,
  ServerlessTriggerError,
  ServerlessTriggerRequest,
} from '../generated/proto/v1/serverless';
import { createHandler, type HandlerOptions, type ServerlessHandler } from '../handler';
import {
  ENDPOINT_ID_HEADER,
  SIGNATURE_HEADER,
  TIMESTAMP_HEADER,
  TRIGGER_ENVELOPE_VERSION,
  namespacePrefix,
} from '../handler/contract';
import { signHex } from '../handler/signature';
import {
  DurableOperator,
  type DurableInvokeOptions,
  type DurableResult,
  type DurableRun,
  type VirtualClock,
} from './durable-operator';

export type {
  DurableInvokeOptions,
  DurableResult,
  DurableRun,
  RecordedFrame,
  VirtualClock,
} from './durable-operator';
export { createSocketPair } from './socket-pair';
export type { SocketPair } from './socket-pair';

export interface TestOperatorOptions extends Omit<
  HandlerOptions,
  'secret' | 'runtime' | 'endpointId'
> {
  /** The endpoint's signing secret. */
  secret: string;
  /** The namespace the endpoint was registered under; a random UUID when omitted. */
  namespace?: string;
  /** The endpoint id sent in X-Hatchet-Endpoint-Id; a random UUID when omitted. */
  endpointId?: string;
  /** The runtime name reported in the healthcheck. Defaults to "test". */
  runtimeName?: string;
  /**
   * Whether the test operator dials durable tasks over the relay, which is what an adapter
   * with an upgrade hook does. Defaults to true; false reproduces an adapter without one.
   */
  durable?: boolean;
}

export interface InvokeOptions {
  retryCount?: number;
  additionalMetadata?: Record<string, string>;
  /** Parent task outputs, for tasks inside a workflow DAG (`ctx.parentOutput`). */
  parents?: Record<string, unknown>;
  workflowRunId?: string;
  taskRunExternalId?: string;
  /** How the run was triggered; defaults to "manual". */
  triggeredBy?: string;
  /** The envelope timestamp in unix seconds; defaults to now. */
  timestamp?: number;
  /** The envelope endpoint id; defaults to the operator's. */
  endpointId?: string;
}

/** What to invoke: a task declaration, an action id (`workflow:task`), or a task of a workflow. */
export type InvokeTarget =
  | TaskWorkflowDeclaration<any, any>
  | string
  | {
      workflow: WorkflowDeclaration<any, any> | BaseWorkflowDeclaration<any, any>;
      task: CreateWorkflowTaskOpts<any, any> | string;
    };

/** The outcome the operator would report for the response, per classifyResponse. */
export type DeliveryOutcome =
  | { status: 'completed'; output: unknown; httpStatus: number }
  | { status: 'failed'; error: string; retry: boolean; httpStatus: number };

/** Thrown by `invoke` when the task did not complete. */
export class InvocationFailedError extends Error {
  constructor(
    readonly error: string,
    readonly retry: boolean,
    readonly httpStatus: number
  ) {
    super(`task failed (${httpStatus}, retry ${retry}): ${error}`);
    this.name = 'InvocationFailedError';
  }
}

export interface TestOperator {
  readonly handler: ServerlessHandler;
  readonly namespace: string;
  readonly endpointId: string;
  /** POSTs a signed healthcheck, validates the body is protojson and returns it decoded. */
  healthcheck(): Promise<ServerlessHealthcheckResponse>;
  /** Signs and POSTs a trigger for the target and returns the task's output. */
  invoke<O = unknown>(target: InvokeTarget, input: unknown, options?: InvokeOptions): Promise<O>;
  /** Like `invoke`, but returns the operator's classification instead of throwing on failure. */
  deliver(target: InvokeTarget, input: unknown, options?: InvokeOptions): Promise<DeliveryOutcome>;
  /** Sends a raw request to the handler; the body is signed unless `sign` is false. */
  request(path: string, init?: RequestInit & { sign?: boolean }): Promise<Response>;
  /**
   * Dials the handler over the durable relay for one invocation and returns its outcome:
   * completed with the output, evicted after an eviction ack, or failed.
   */
  invokeDurable(
    target: InvokeTarget,
    input: unknown,
    options?: DurableInvokeOptions
  ): Promise<DurableResult>;
  /** Like `invokeDurable`, but returns the invocation in progress for mid-flight actions. */
  startDurable(target: InvokeTarget, input: unknown, options?: DurableInvokeOptions): DurableRun;
  /** Re-invokes an evicted invocation against the same event log with the next count. */
  resume(previous: DurableResult, options?: DurableInvokeOptions): Promise<DurableResult>;
  /** The virtual clock durable sleeps are measured against. */
  clock: VirtualClock;
  /** Delivers a user event to every durable wait on the key. */
  emit(eventKey: string, payload?: Record<string, unknown>): void;
  /** Signed upgrade headers for the task and invocation, with optional overrides. */
  upgradeHeaders: DurableOperator['upgradeHeaders'];
}

const ORIGIN = 'https://endpoint.test';

export function createTestOperator(options: TestOperatorOptions): TestOperator {
  const {
    secret,
    namespace = crypto.randomUUID(),
    endpointId = crypto.randomUUID(),
    runtimeName = 'test',
    durable = true,
    ...rest
  } = options;
  const handler = createHandler({
    ...rest,
    secret,
    endpointId,
    durable,
    runtime: { name: runtimeName },
  });
  const prefix = namespacePrefix(namespace);

  const request: TestOperator['request'] = async (path, init = {}) => {
    const { sign = true, ...requestInit } = init;

    const method = (requestInit.method ?? 'POST').toUpperCase();

    if (requestInit.body === undefined && method !== 'GET' && method !== 'HEAD') {
      // A fresh healthcheck-shaped body, so a raw request passes the freshness checks.
      requestInit.body = JSON.stringify(
        ServerlessHealthcheckRequest.toJSON({
          endpointId,
          namespace,
          timestamp: Math.floor(Date.now() / 1000),
        })
      );
    }

    const body = typeof requestInit.body === 'string' ? requestInit.body : '';
    const headers = new Headers(requestInit.headers);

    if (sign) {
      headers.set(SIGNATURE_HEADER, await signHex(secret, body));
      headers.set(ENDPOINT_ID_HEADER, endpointId);
      headers.set(TIMESTAMP_HEADER, String(Math.floor(Date.now() / 1000)));
    }

    return handler.fetch(
      new Request(`${ORIGIN}${path}`, { method: 'POST', ...requestInit, headers })
    );
  };

  const deliver: TestOperator['deliver'] = async (target, input, invokeOptions = {}) => {
    const { workflowName, taskName } = resolveTarget(target);
    const timestamp = invokeOptions.timestamp ?? Math.floor(Date.now() / 1000);

    const action = AssignedAction.fromPartial({
      tenantId: crypto.randomUUID(),
      workflowRunId: invokeOptions.workflowRunId ?? crypto.randomUUID(),
      jobId: crypto.randomUUID(),
      jobName: `${prefix}${workflowName}`,
      jobRunId: crypto.randomUUID(),
      taskId: crypto.randomUUID(),
      taskRunExternalId: invokeOptions.taskRunExternalId ?? crypto.randomUUID(),
      actionId: createActionId(`${prefix}${workflowName}`, taskName),
      actionType: ActionType.START_STEP_RUN,
      actionPayload: JSON.stringify({
        input,
        parents: invokeOptions.parents ?? {},
        triggered_by: invokeOptions.triggeredBy ?? 'manual',
      }),
      taskName,
      retryCount: invokeOptions.retryCount ?? 0,
      additionalMetadata: invokeOptions.additionalMetadata
        ? JSON.stringify(invokeOptions.additionalMetadata)
        : undefined,
      priority: 1,
    });

    const envelope = ServerlessTriggerRequest.toJSON({
      endpointId: invokeOptions.endpointId ?? endpointId,
      namespace,
      action,
      timestamp,
      version: TRIGGER_ENVELOPE_VERSION,
    });

    const response = await request(`${handler.basePath}/trigger`, {
      body: JSON.stringify(envelope),
      headers: { 'content-type': 'application/json' },
    });

    return classifyResponse(response);
  };

  const durableOperator = new DurableOperator({
    handler,
    secret,
    namespace,
    endpointId,
    runChild: async (workflowName, input) => {
      const workflow = handler
        .healthcheck()
        .workflows.find((candidate) => candidate.name === workflowName);

      if (!workflow || workflow.tasks.length !== 1) {
        throw new Error(
          `child workflow "${workflowName}" must be a single non-durable task to run in the test operator`
        );
      }

      const outcome = await deliver(workflow.tasks[0].action, input);

      if (outcome.status === 'failed') {
        throw new Error(outcome.error);
      }

      return outcome.output;
    },
  });

  const startDurable: TestOperator['startDurable'] = (target, input, durableOptions) => {
    const { workflowName, taskName } = resolveTarget(target);
    return durableOperator.start(workflowName, taskName, input, durableOptions);
  };

  // What each durable task run was started with, so `resume` re-sends the same action.
  const started = new Map<string, { target: InvokeTarget; input: unknown }>();

  return {
    handler,
    namespace,
    endpointId,
    clock: durableOperator.clock,
    emit: (eventKey, payload) => durableOperator.emit(eventKey, payload),
    upgradeHeaders: (taskRunExternalId, invocationCount, overrides) =>
      durableOperator.upgradeHeaders(taskRunExternalId, invocationCount, overrides),

    async healthcheck() {
      const body = JSON.stringify(
        ServerlessHealthcheckRequest.toJSON({
          endpointId,
          namespace,
          timestamp: Math.floor(Date.now() / 1000),
        })
      );
      const response = await request(`${handler.basePath}/healthcheck`, {
        body,
        headers: { 'content-type': 'application/json' },
      });

      if (response.status !== 200) {
        throw new Error(`healthcheck returned status ${response.status}: ${await response.text()}`);
      }

      const raw: unknown = JSON.parse(await response.text());
      const decoded = ServerlessHealthcheckResponse.fromJSON(raw);
      const canonical = ServerlessHealthcheckResponse.toJSON(decoded);

      if (!deepEqual(raw, canonical)) {
        throw new Error(
          `healthcheck body is not canonical protojson.\nreceived: ${JSON.stringify(raw)}\ncanonical: ${JSON.stringify(canonical)}`
        );
      }

      return decoded;
    },

    deliver,

    async invoke(target, input, invokeOptions) {
      const outcome = await deliver(target, input, invokeOptions);

      if (outcome.status === 'failed') {
        throw new InvocationFailedError(outcome.error, outcome.retry, outcome.httpStatus);
      }

      return outcome.output as never;
    },

    request,

    startDurable(target, input, durableOptions) {
      const run = startDurable(target, input, durableOptions);
      started.set(run.taskRunExternalId, { target, input });
      return run;
    },

    invokeDurable(target, input, durableOptions) {
      return this.startDurable(target, input, durableOptions).result;
    },

    resume(previous, durableOptions = {}) {
      const origin = started.get(previous.taskRunExternalId);

      if (!origin) {
        throw new Error(`no durable invocation known for task ${previous.taskRunExternalId}`);
      }

      return this.invokeDurable(origin.target, origin.input, {
        ...durableOptions,
        taskRunExternalId: previous.taskRunExternalId,
        invocationCount: previous.invocationCount + 1,
      });
    },
  };
}

function resolveTarget(target: InvokeTarget): { workflowName: string; taskName: string } {
  if (typeof target === 'string') {
    const separator = target.lastIndexOf(':');

    if (separator === -1) {
      throw new Error(`action id "${target}" must look like "workflow:task"`);
    }

    return { workflowName: target.slice(0, separator), taskName: target.slice(separator + 1) };
  }

  if ('workflow' in target && 'task' in target) {
    return {
      workflowName: target.workflow.definition.name,
      taskName: typeof target.task === 'string' ? target.task : target.task.name,
    };
  }

  const { definition } = target;
  const task = definition._tasks[0] ?? definition._durableTasks[0];

  if (!task) {
    throw new Error(`workflow "${definition.name}" declares no task`);
  }

  return { workflowName: definition.name, taskName: task.name };
}

/**
 * Mirrors pkg/serverlessoperator/delivery.go classifyResponse: 204 completes with no
 * output, 2xx JSON is the output, 4xx is permanent except 408, 425 and 429, 5xx is
 * retryable, and a ServerlessTriggerError body overrides message and retry on non-2xx.
 */
export async function classifyResponse(response: Response): Promise<DeliveryOutcome> {
  const httpStatus = response.status;
  const text = await response.text();

  if (httpStatus === 204) {
    return { status: 'completed', output: undefined, httpStatus };
  }

  if (httpStatus >= 200 && httpStatus < 300) {
    if (text.length === 0) {
      return { status: 'completed', output: undefined, httpStatus };
    }

    try {
      return { status: 'completed', output: JSON.parse(text), httpStatus };
    } catch {
      return {
        status: 'failed',
        error: `endpoint returned status ${httpStatus} with a non-JSON body`,
        retry: false,
        httpStatus,
      };
    }
  }

  let retry = httpStatus >= 500 || httpStatus === 408 || httpStatus === 425 || httpStatus === 429;
  let error = `endpoint returned status ${httpStatus}`;

  if (httpStatus >= 300 && httpStatus < 400) {
    error = `endpoint returned redirect status ${httpStatus}; redirects are not followed`;
  }

  if (text.length > 0) {
    try {
      const override = ServerlessTriggerError.fromJSON(JSON.parse(text));

      if (override.error) {
        ({ error } = override);
      }

      if (override.retry !== undefined) {
        ({ retry } = override);
      }
    } catch {
      // Not a ServerlessTriggerError body; the status mapping stands.
    }
  }

  return { status: 'failed', error, retry, httpStatus };
}

function deepEqual(a: unknown, b: unknown): boolean {
  if (a === b) {
    return true;
  }

  if (Array.isArray(a) || Array.isArray(b)) {
    return (
      Array.isArray(a) &&
      Array.isArray(b) &&
      a.length === b.length &&
      a.every((item, index) => deepEqual(item, b[index]))
    );
  }

  if (typeof a !== 'object' || typeof b !== 'object' || a === null || b === null) {
    return false;
  }

  const left = a as Record<string, unknown>;
  const right = b as Record<string, unknown>;
  const keys = new Set([...Object.keys(left), ...Object.keys(right)]);

  for (const key of keys) {
    if (!deepEqual(left[key], right[key])) {
      return false;
    }
  }

  return true;
}
