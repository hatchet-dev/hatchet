import type { Transport } from '@connectrpc/connect';
import HatchetError from '@util/errors/hatchet-error';
import { IdempotencyCollisionError } from '@util/errors/idempotency-collision-error';
import { BulkTriggerPartialError } from '@util/errors/bulk-trigger-partial-error';
import {
  createFetchTransport,
  resolveServerUrl as resolveFetchServerUrl,
} from '@clients/transport/fetch-transport';
import { createV1AdminRpc, createWorkflowsRpc } from '@hatchet/clients/admin/rpc';
import {
  buildTriggerWorkflowRequest,
  extractBulkTriggerCollision,
  extractExistingRunId,
  isAlreadyExists,
} from '@hatchet/clients/admin/trigger-request';
import { createEventsRpc } from '@hatchet/clients/event/rpc';
import type { EventsServiceClient } from '@hatchet/protoc/events/events';
import {
  BulkTriggerWorkflowRequest,
  type BulkTriggerWorkflowResponse,
  type WorkflowServiceClient,
} from '@hatchet/protoc/workflows';
import type { AdminServiceClient } from '@hatchet/protoc/v1/workflows';
import {
  getGrpcBroadcastAddressFromJWT,
  getTenantIdFromJWT,
} from '@hatchet/util/config-loader/token';
import { batch } from '@hatchet/util/batch';
import { retrier } from '@hatchet/util/retrier';
import type { Logger } from '@hatchet/util/logger/logger';
import {
  BaseWorkflowDeclaration,
  TaskWorkflowDeclaration,
  type RunManyOpt,
  type RunOpts,
} from '@hatchet/v1/declaration';
import type { InputType, OutputType, UnknownInputType } from '@hatchet/v1/types';
import { EventsClient } from './features/events';
import { LogsClient } from './features/logs';
import { RateLimitsClient } from './features/rate-limits';
import { RunsClient } from './features/runs';
import { StreamsClient } from './features/streams';
import { WorkflowsClient } from './features/workflows';
import { consoleLogger } from './logger';
import { WorkflowRunRef } from './run-ref';
import type { CoreClientConfig, TriggerRunOptions } from './types';

const BULK_TRIGGER_BATCH_SIZE = 500;
const BULK_TRIGGER_MAX_BYTES = 4 * 1024 * 1024;

/** A workflow to run: a declaration from `/edge`, anything with a `name`, or the name itself. */
export type WorkflowRef<I extends InputType = UnknownInputType, O extends OutputType = void> =
  BaseWorkflowDeclaration<I, O> | { name: string } | string;

/**
 * The resolved client configuration: what the client was built with, plus the values derived
 * from it.
 */
export interface ResolvedCoreConfig {
  serverUrl: string;
  tenantId: string;
  namespace: string;
  logLevel: CoreClientConfig['logLevel'];
  retrier: CoreClientConfig['retrier'];
}

/**
 * The application client for runtimes without gRPC: it triggers runs, pushes events, waits
 * for results and manages workflows over unary Connect calls on a fetch transport, so it runs
 * in Cloudflare Workers, Vercel Functions, Deno, Bun, browsers and Node alike.
 *
 * It is configured from an explicit object only. Nothing is read from the environment or a
 * config file, and no worker can be started from it; workers stay on the Node client.
 *
 * ```typescript
 * import { Hatchet } from '@hatchet-dev/typescript-sdk/core';
 *
 * const hatchet = new Hatchet({ token: env.HATCHET_CLIENT_TOKEN });
 * const output = await hatchet.run('greet', { name: 'world' });
 * ```
 */
export class HatchetCore {
  readonly config: ResolvedCoreConfig;
  /** The transport every call goes through. */
  readonly transport: Transport;
  readonly logger: Logger;
  /** Pushes events. */
  readonly events: EventsClient;
  /** Gets, cancels and replays runs. */
  readonly runs: RunsClient;
  /** Registers workflows. */
  readonly workflows: WorkflowsClient;
  /** Creates and updates rate limits. */
  readonly rateLimits: RateLimitsClient;
  /** Writes task run log lines. */
  readonly logs: LogsClient;
  /** Publishes task run stream chunks. */
  readonly streams: StreamsClient;

  private readonly workflowsRpc: WorkflowServiceClient;
  private readonly adminRpc: AdminServiceClient;
  private readonly eventsRpc: EventsServiceClient;

  constructor(config: CoreClientConfig) {
    if (!config.token) {
      throw new HatchetError('a token is required to create a Hatchet client');
    }

    const tenantId = getTenantIdFromJWT(config.token);
    const serverUrl = resolveServerUrl(config);
    const namespace = normalizeNamespace(config.namespace);
    const logger = config.logger ?? consoleLogger;

    this.config = {
      serverUrl,
      tenantId,
      namespace,
      logLevel: config.logLevel,
      retrier: config.retrier,
    };
    this.logger = logger('HatchetCore', config.logLevel);
    this.transport =
      config.transport ??
      createFetchTransport({ token: config.token, serverUrl, tls: config.tls, fetch: config.fetch });

    this.workflowsRpc = createWorkflowsRpc(this.transport);
    this.adminRpc = createV1AdminRpc(this.transport);
    this.eventsRpc = createEventsRpc(this.transport);

    const shared = { logger: this.logger, namespace, retrier: config.retrier };
    this.events = new EventsClient({ rpc: this.eventsRpc, ...shared });
    this.runs = new RunsClient(this.adminRpc);
    this.workflows = new WorkflowsClient({ rpc: this.adminRpc, ...shared });
    this.rateLimits = new RateLimitsClient({ rpc: this.workflowsRpc, ...shared });
    this.logs = new LogsClient(this.eventsRpc);
    this.streams = new StreamsClient(this.eventsRpc);
  }

  /** The same as `new HatchetCore(config)`, mirroring the Node client's `init`. */
  static init(config: CoreClientConfig): HatchetCore {
    return new HatchetCore(config);
  }

  /** The tenant the token belongs to. */
  get tenantId(): string {
    return this.config.tenantId;
  }

  /**
   * Triggers a run and returns a reference to it without waiting.
   *
   * The options are the Node client's `admin.runWorkflow` options and go to the engine as
   * is: the workflow name is namespaced and lowercased and the input is JSON. A task
   * declaration's reference resolves to the task's own output.
   */
  async runNoWait<I extends InputType = UnknownInputType, O extends OutputType = void>(
    workflow: WorkflowRef<I, O>,
    input: I,
    options: TriggerRunOptions = {}
  ): Promise<WorkflowRunRef<O>> {
    const request = buildTriggerWorkflowRequest(
      workflowName(workflow),
      input,
      options,
      this.config.namespace
    );

    try {
      const response = await retrier(
        () => this.workflowsRpc.triggerWorkflow(request),
        this.logger,
        { ...this.config.retrier, shouldRetry: (e) => !isAlreadyExists(e) }
      );
      return new WorkflowRunRef<O>(
        response.workflowRunId,
        this.runs,
        options.parentId,
        options._standaloneTaskName ?? standaloneTaskName(workflow)
      );
    } catch (e: unknown) {
      if (isAlreadyExists(e)) {
        throw new IdempotencyCollisionError(extractExistingRunId(e));
      }
      throw new HatchetError(e instanceof Error ? e.message : String(e));
    }
  }

  /** Triggers a run and waits for its output; see `WorkflowRunRef.result()` for the wait. */
  async run<I extends InputType = UnknownInputType, O extends OutputType = void>(
    workflow: WorkflowRef<I, O>,
    input: I,
    options: TriggerRunOptions = {}
  ): Promise<O> {
    const ref = await this.runNoWait<I, O>(workflow, input, options);
    return ref.result();
  }

  /**
   * Triggers many runs of one workflow in batches of 500 (or 4 MB) over `BulkTriggerWorkflow`
   * and returns their references in input order. `runs` takes the `RunManyOpt` entries the
   * declarations' `runMany` takes.
   *
   * Each batch is one call, so a failure part way through leaves the earlier batches' runs
   * created. That failure rejects with a `BulkTriggerPartialError` whose `refs` are those runs'
   * references and whose `cause` is the batch's own error, so the caller can keep or cancel
   * them instead of retrying the whole request and creating them twice. A failure of the
   * first batch rejects with the batch's error itself: a
   * `BulkTriggerIdempotencyCollisionError` on an idempotency key collision, otherwise a
   * `HatchetError`.
   */
  async runManyNoWait<I extends InputType = UnknownInputType, O extends OutputType = void>(
    workflow: WorkflowRef<I, O>,
    runs: RunManyOpt<I>[]
  ): Promise<WorkflowRunRef<O>[]> {
    const name = workflowName(workflow);
    const taskName = standaloneTaskName(workflow);
    const requests = runs.map((run) =>
      buildTriggerWorkflowRequest(
        name,
        run.input,
        runOptsToTrigger(run.opts),
        this.config.namespace
      )
    );
    const batches = batch(requests, BULK_TRIGGER_BATCH_SIZE, BULK_TRIGGER_MAX_BYTES);
    const refs: WorkflowRunRef<O>[] = [];

    for (const { batchIndex, payloads } of batches) {
      const request = BulkTriggerWorkflowRequest.create({ workflows: payloads });
      let response: BulkTriggerWorkflowResponse;
      try {
        response = await retrier(
          () => this.workflowsRpc.bulkTriggerWorkflow(request),
          this.logger,
          { ...this.config.retrier, shouldRetry: (e) => !isAlreadyExists(e) }
        );
      } catch (e: unknown) {
        const error = bulkTriggerError(e);
        throw batchIndex === 0 ? error : new BulkTriggerPartialError(refs, batchIndex, error);
      }
      refs.push(
        ...response.workflowRunIds.map(
          (id) => new WorkflowRunRef<O>(id, this.runs, undefined, taskName)
        )
      );
    }
    return refs;
  }

  /**
   * Triggers many runs of one workflow and waits for all of their outputs, in input order.
   *
   * When any entry sets `opts.returnExceptions`, a failed run does not reject the whole
   * wait: its slot holds an `Error` instead (a task error list is joined with `; `), the way
   * a declaration's `runMany` behaves on the Node client.
   */
  async runMany<I extends InputType = UnknownInputType, O extends OutputType = void>(
    workflow: WorkflowRef<I, O>,
    runs: RunManyOpt<I>[]
  ): Promise<O[]> {
    const refs = await this.runManyNoWait<I, O>(workflow, runs);
    const results = refs.map((ref) => ref.result());

    if (runs.some((run) => run.opts?.returnExceptions)) {
      const settled = await Promise.allSettled(results);
      return settled.map((s) => (s.status === 'fulfilled' ? s.value : asError(s.reason))) as O[];
    }
    return Promise.all(results);
  }

  /** A reference to an existing run, to wait on, cancel or replay it. */
  runRef<T = unknown>(id: string): WorkflowRunRef<T> {
    return this.runs.runRef<T>(id);
  }
}

function workflowName<I extends InputType, O extends OutputType>(
  workflow: WorkflowRef<I, O>
): string {
  return typeof workflow === 'string' ? workflow : workflow.name;
}

function standaloneTaskName<I extends InputType, O extends OutputType>(
  workflow: WorkflowRef<I, O>
): string | undefined {
  return workflow instanceof TaskWorkflowDeclaration ? workflow._standalone_task_name : undefined;
}

/**
 * The error one bulk trigger batch raises: the collision details when the engine attached
 * them, otherwise a `HatchetError`.
 */
function bulkTriggerError(e: unknown): Error {
  if (isAlreadyExists(e)) {
    const collision = extractBulkTriggerCollision(e);
    if (collision) return collision;
  }
  return new HatchetError(e instanceof Error ? e.message : String(e));
}

/**
 * The rejection of one run as `runMany` returns it under `returnExceptions`: an `Error` as is,
 * a task error list joined with `; `, anything else as its string; the Node declaration's
 * normalization.
 */
function asError(reason: unknown): Error {
  if (reason instanceof Error) return reason;
  return new Error(Array.isArray(reason) ? reason.join('; ') : String(reason));
}

/**
 * Maps a declaration's `RunOpts` onto the trigger options. `sticky` only means something
 * inside a worker task and is dropped; `returnExceptions` is read by `runMany` itself.
 */
function runOptsToTrigger(opts: RunOpts | undefined): TriggerRunOptions | undefined {
  if (!opts) return undefined;
  return {
    additionalMetadata: opts.additionalMetadata,
    priority: opts.priority,
    childKey: opts.childKey,
    desiredWorkerLabels: opts.desiredWorkerLabels,
  };
}

/**
 * The engine's base URL: `serverUrl` checked against the TLS strategy, `hostPort` with the
 * TLS strategy, or the `grpc_broadcast_address` the token was issued with.
 */
function resolveServerUrl(config: CoreClientConfig): string {
  if (config.serverUrl || config.hostPort) {
    return resolveFetchServerUrl(config);
  }

  const hostPort = getGrpcBroadcastAddressFromJWT(config.token);
  if (!hostPort) {
    throw new HatchetError(
      'the token carries no grpc_broadcast_address claim; set serverUrl or hostPort on the client config'
    );
  }
  return resolveFetchServerUrl({ hostPort, tls: config.tls });
}

/** Namespaces end in an underscore and are lowercase, as the Node client's config loader makes them. */
function normalizeNamespace(namespace: string | undefined): string {
  if (!namespace) return '';
  return (namespace.endsWith('_') ? namespace : `${namespace}_`).toLowerCase();
}
