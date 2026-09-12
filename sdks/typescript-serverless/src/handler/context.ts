/**
 * The `ContextRuntime` a serverless task runs under. A worker implements the same interface
 * over its Hatchet client; here there is no client, so every engine-facing operation throws
 * `ServerlessLimitationError` and logging goes to the console.
 */
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
import { ServerlessLimitationError } from './errors';

export type ConsoleLike = Pick<Console, 'debug' | 'info' | 'warn' | 'error'>;

export interface ServerlessRuntimeOptions {
  /** The namespace the operator registered the workflows under, with its separator. */
  namespace?: string;
  /** Whether the handler serves the given workflow (un-namespaced name). */
  hasWorkflow: (workflowName: string) => boolean;
  /** Where task logs go; defaults to the global console. */
  console?: ConsoleLike;
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

export class ServerlessRuntime implements ContextRuntime {
  readonly namespace?: string;

  constructor(private readonly options: ServerlessRuntimeOptions) {
    this.namespace = options.namespace;
  }

  logger(name: string): Logger {
    return new ConsoleLogger(name, this.options.console);
  }

  /** `ctx.log` and `ctx.logger.*` print to the console; nothing reaches the engine. */
  async putLog(
    _taskRunExternalId: string,
    _message: string,
    _level: LogLevel | undefined,
    _retryCount: number,
    _extra?: Record<string, unknown>
  ): Promise<void> {}

  async cancelRun(_taskRunExternalId: string): Promise<void> {
    throw new ServerlessLimitationError('ctx.cancel');
  }

  async cancelBatch(_request: CancelBatchRequest): Promise<void> {
    throw new ServerlessLimitationError('ctx.cancel');
  }

  async refreshTimeout(_taskRunExternalId: string, _incrementBy: string): Promise<void> {
    throw new ServerlessLimitationError('ctx.refreshTimeout');
  }

  async releaseSlot(_taskRunExternalId: string): Promise<void> {
    throw new ServerlessLimitationError(
      'ctx.releaseSlot',
      'A serverless endpoint has no worker slots'
    );
  }

  async putStream(
    _taskRunExternalId: string,
    _data: string | Uint8Array,
    _index: number
  ): Promise<void> {
    throw new ServerlessLimitationError('ctx.putStream');
  }

  async runWorkflow<Q = object>(
    _workflowName: string,
    _input: Q,
    _options?: SpawnRunOptions
  ): Promise<never> {
    throw new ServerlessLimitationError(
      'ctx.runChild (and ctx.runNoWaitChild, ctx.spawnWorkflow)',
      'Durable tasks will be able to spawn child runs over the operator relay in a later version'
    );
  }

  async runWorkflows<Q = object>(_runs: SpawnRunRequest<Q>[]): Promise<never> {
    throw new ServerlessLimitationError(
      'ctx.bulkRunChildren (and ctx.bulkRunNoWaitChildren, ctx.spawnWorkflows)',
      'Durable tasks will be able to spawn child runs over the operator relay in a later version'
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
    throw new ServerlessLimitationError('ctx.worker.labels', 'A serverless endpoint has no worker');
  }

  async upsertWorkerLabels(_labels: WorkerLabels): Promise<WorkerLabels> {
    throw new ServerlessLimitationError(
      'ctx.worker.upsertLabels',
      'A serverless endpoint has no worker'
    );
  }
}
