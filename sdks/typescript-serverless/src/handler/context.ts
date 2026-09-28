/**
 * The `ContextRuntime` a serverless task runs under. A worker implements the same interface
 * over its Hatchet client; here the engine-facing operations go to the handler's optional
 * core client (unary calls), child results are awaited on the invocation socket, and logging
 * goes to the console as well. Without a client every engine-facing operation throws
 * `ServerlessLimitationError`.
 */
import type { LogLevel as EventLogLevel } from '@hatchet-dev/typescript-sdk/core/index.js';
import type {
  CancelBatchRequest,
  ContextRuntime,
  LogExtra,
  LogLevel,
  Logger,
  SpawnRunOptions,
  SpawnRunRequest,
  WorkerLabels,
} from '@hatchet-dev/typescript-sdk/edge/index.js';
import type NodeWorkflowRunRef from '@hatchet-dev/typescript-sdk/util/workflow-run-ref.js';
import type { TriggerRunOptions, WorkflowRunRef } from '@hatchet-dev/typescript-sdk/core/index.js';
import type { ClientSource, HatchetCore } from './client';
import { ServerlessLimitationError } from './errors';
import { errorMessage } from './http';
import { TaskRunRef, type RunWatcher } from './runs';

export type ConsoleLike = Pick<Console, 'debug' | 'info' | 'warn' | 'error'>;

export interface ServerlessRuntimeOptions {
  /** Whether the handler serves the given workflow, by the name it was declared with. */
  hasWorkflow: (workflowName: string) => boolean;
  /** Where task logs go; defaults to the global console. */
  console?: ConsoleLike;
  /** Hands out the handler's client, building it on first use; absent without a `client` option. */
  client?: ClientSource;
  /** Awaits child runs on the invocation socket; absent when the task was invoked over a POST. */
  runs?: RunWatcher;
  /**
   * The task is a durable invocation: `ctx.runChild` and its variants throw, since a durable
   * task spawns children through the durable log (`ctx.spawnChild`), which replays them.
   */
  durable?: boolean;
}

/** A `Logger` over the console. */
export class ConsoleLogger implements Logger {
  constructor(
    private readonly name: string,
    private readonly out: ConsoleLike = console
  ) {}

  debug(message: string, extra?: LogExtra) {
    this.out.debug(...this.line(message, extra));
  }

  info(message: string, extra?: LogExtra) {
    this.out.info(...this.line(message, extra));
  }

  green(message: string, extra?: LogExtra) {
    this.out.info(...this.line(message, extra));
  }

  warn(message: string, error?: Error, extra?: LogExtra) {
    this.out.warn(...this.line(message, extra, error));
  }

  error(message: string, error?: Error, extra?: LogExtra) {
    this.out.error(...this.line(message, extra, error));
  }

  util(_key: string, message: string, extra?: LogExtra) {
    this.out.debug(...this.line(message, extra));
  }

  private line(message: string, extra?: LogExtra, error?: Error): unknown[] {
    const parts: unknown[] = [`[hatchet:${this.name}] ${message}`];

    if (extra && Object.keys(extra).length > 0) {
      parts.push(extra);
    }

    if (error) {
      parts.push(error);
    }

    return parts;
  }
}

const CHILD_RUN_FEATURES =
  'ctx.runChild (and ctx.runNoWaitChild, ctx.bulkRunChildren, ctx.bulkRunNoWaitChildren, ctx.spawnWorkflow, ctx.spawnWorkflows)';

/** The core client's log level for a context log level; `OFF` and none mean the default. */
function eventLogLevel(level: LogLevel | undefined): EventLogLevel | undefined {
  if (level === undefined || level === 'OFF') {
    return undefined;
  }

  return level as unknown as EventLogLevel;
}

/** Maps a context's spawn options onto the trigger options the core client sends. */
function triggerOptions(options: SpawnRunOptions | undefined): TriggerRunOptions {
  if (!options) {
    return {};
  }

  return {
    parentId: options.parentId,
    parentTaskRunExternalId: options.parentTaskRunExternalId,
    parentStepRunId: options.parentStepRunId,
    childIndex: options.childIndex,
    childKey: options.childKey ?? options.key,
    additionalMetadata: options.additionalMetadata,
    desiredWorkerId: options.desiredWorkerId,
    priority: options.priority,
    desiredWorkerLabels: options.desiredWorkerLabels,
    _standaloneTaskName: options._standaloneTaskName,
  };
}

export class ServerlessRuntime implements ContextRuntime {
  private readonly out: ConsoleLike;

  constructor(private readonly options: ServerlessRuntimeOptions) {
    this.out = options.console ?? console;
  }

  logger(name: string): Logger {
    return new ConsoleLogger(name, this.options.console);
  }

  /** The handler's client, or a `ServerlessLimitationError` for the feature without one. */
  private client(feature: string, detail?: string): HatchetCore {
    const client = this.options.client?.();

    if (!client) {
      throw new ServerlessLimitationError(feature, detail);
    }

    return client;
  }

  /**
   * `ctx.log` prints through the context's logger; with a client the line is also written to
   * the engine, and a write that fails is reported on the console rather than failing the
   * task, as the Node client's fire-and-forget `putLog` behaves.
   */
  async putLog(
    taskRunExternalId: string,
    message: string,
    level: LogLevel | undefined,
    retryCount: number,
    extra?: Record<string, unknown>
  ): Promise<void> {
    const client = this.options.client?.();

    if (!client) {
      return;
    }

    try {
      await client.logs.put(taskRunExternalId, message, {
        level: eventLogLevel(level),
        retryCount,
        metadata: extra,
      });
    } catch (err) {
      this.out.warn(`[hatchet] could not write a log line to the engine: ${errorMessage(err)}`);
    }
  }

  async cancelRun(taskRunExternalId: string): Promise<void> {
    await this.client('ctx.cancel').runs.cancel({ ids: [taskRunExternalId] });
  }

  async cancelBatch(_request: CancelBatchRequest): Promise<void> {
    throw new ServerlessLimitationError(
      'ctx.cancel on a batch',
      undefined,
      'batch tasks are not supported on a serverless endpoint'
    );
  }

  async refreshTimeout(_taskRunExternalId: string, _incrementBy: string): Promise<void> {
    throw new ServerlessLimitationError(
      'ctx.refreshTimeout',
      "Set the task's executionTimeout and the endpoint's requestTimeoutSeconds instead",
      'extending a running task is a worker call'
    );
  }

  async releaseSlot(_taskRunExternalId: string): Promise<void> {
    throw new ServerlessLimitationError(
      'ctx.releaseSlot',
      undefined,
      'a serverless endpoint has no worker slots'
    );
  }

  async putStream(
    taskRunExternalId: string,
    data: string | Uint8Array,
    index: number
  ): Promise<void> {
    await this.client('ctx.putStream').streams.put(taskRunExternalId, data, index);
  }

  async runWorkflow<Q = object, P = object>(
    workflowName: string,
    input: Q,
    options?: SpawnRunOptions
  ): Promise<NodeWorkflowRunRef<P>> {
    this.assertNotDurable();

    const client = this.client(CHILD_RUN_FEATURES);
    const ref = await client.runNoWait(workflowName, input as never, triggerOptions(options));

    return this.taskRef(client, ref) as unknown as NodeWorkflowRunRef<P>;
  }

  /**
   * Triggers every run with its own call, in parallel: each child carries its own parent,
   * index and key, which the core client's bulk trigger (one workflow, shared options) does
   * not express.
   */
  async runWorkflows<Q = object, P = object>(
    runs: SpawnRunRequest<Q>[]
  ): Promise<NodeWorkflowRunRef<P>[]> {
    this.assertNotDurable();

    const client = this.client(CHILD_RUN_FEATURES);
    const refs = await Promise.all(
      runs.map((run) =>
        client.runNoWait(run.workflowName, run.input as never, triggerOptions(run.options))
      )
    );

    return refs.map((ref) => this.taskRef(client, ref) as unknown as NodeWorkflowRunRef<P>);
  }

  private assertNotDurable(): void {
    if (this.options.durable) {
      throw new ServerlessLimitationError(
        CHILD_RUN_FEATURES,
        'Use ctx.spawnChild and ctx.spawnChildren, which record the child in the durable event log and replay its result after an eviction',
        'a durable task spawns children through the durable log'
      );
    }
  }

  private taskRef(client: HatchetCore, ref: WorkflowRunRef<unknown>): TaskRunRef<unknown> {
    return new TaskRunRef<unknown>(
      ref.workflowRunId,
      client.runs,
      this.options.runs,
      ref.parentWorkflowRunId,
      ref._standaloneTaskName
    );
  }

  /** There is no worker, so no worker id. */
  workerId(): string | undefined {
    return undefined;
  }

  hasWorkflow(workflowName: string): boolean {
    return this.options.hasWorkflow(workflowName);
  }

  workerLabels(): WorkerLabels {
    throw new ServerlessLimitationError(
      'ctx.worker.labels',
      undefined,
      'a serverless endpoint has no worker'
    );
  }

  async upsertWorkerLabels(_labels: WorkerLabels): Promise<WorkerLabels> {
    throw new ServerlessLimitationError(
      'ctx.worker.upsertLabels',
      undefined,
      'a serverless endpoint has no worker'
    );
  }
}
