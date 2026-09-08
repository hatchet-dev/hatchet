/**
 * The seams between a task's `Context` / `DurableContext` and whatever runs the task.
 *
 * A worker process implements them over its `HatchetClient` and durable listener (see
 * `worker-runtime.ts`); a serverless handler implements them over the operator's request
 * and durable socket. Nothing in this module imports from Node, so the interfaces are
 * usable from the edge entry point.
 * @module Runtime
 */
import type { Logger, LogLevel } from '@hatchet/util/logger';
import type WorkflowRunRef from '@hatchet/util/workflow-run-ref';
import type { WorkerLabelComparator } from '@hatchet/protoc/v1/shared/trigger';
import type { Priority } from '@hatchet/v1/declaration';
import type {
  DurableTaskEventLogEntryResult,
  DurableTaskEventMemoAck,
  DurableTaskEventRunAck,
  DurableTaskEventWaitForAck,
  MemoEvent,
  RunChildrenEvent,
  WaitForEvent,
} from '@hatchet/clients/listeners/durable-listener/durable-events';

/** Labels a worker advertises for affinity-based assignment. */
export type WorkerLabels = Record<string, string | number | undefined>;

/** A worker label a spawned child run asks for. */
export type DesiredWorkerLabel = {
  value: string | number;
  required?: boolean;
  weight?: number;
  comparator?: WorkerLabelComparator;
};

/**
 * Options a context passes when it spawns a child run. Mirrors the options accepted by
 * `AdminClient.runWorkflow`.
 */
export type SpawnRunOptions = {
  parentId?: string | undefined;
  parentTaskRunExternalId?: string | undefined;
  /** @deprecated Use `parentTaskRunExternalId` instead. */
  parentStepRunId?: string | undefined;
  childIndex?: number | undefined;
  childKey?: string | undefined;
  /** Alias of `childKey` kept for the child run APIs on `Context`. */
  key?: string | undefined;
  additionalMetadata?: Record<string, string> | undefined;
  desiredWorkerId?: string | undefined;
  priority?: Priority;
  sticky?: boolean;
  returnExceptions?: boolean;
  desiredWorkerLabels?: Record<string, DesiredWorkerLabel>;
  _standaloneTaskName?: string | undefined;
};

export type SpawnRunRequest<Q = object> = {
  workflowName: string;
  input: Q;
  options?: SpawnRunOptions;
};

/** Identifies every member of a batch task run when the whole batch is cancelled. */
export type CancelBatchRequest = {
  workerId: string;
  jobId: string;
  actionId: string;
  batchId: string;
  memberIds: string[];
};

/**
 * Everything a `Context` needs from the process running the task: a logger factory,
 * the namespace, the engine-facing operations, and the worker facts `ctx.worker` reports.
 *
 * Implementations that cannot honour an operation (for example a serverless runtime
 * without a Hatchet client) throw from it; the context does not guard against that.
 */
export interface ContextRuntime {
  /** The namespace applied to workflow names and event keys, if any. */
  readonly namespace?: string;

  /** Creates a logger for the given component, at the runtime's configured level. */
  logger(name: string): Logger;

  /** Cancels a single task run. */
  cancelRun(taskRunExternalId: string): Promise<void>;

  /** Cancels every member of a batch task run. */
  cancelBatch(request: CancelBatchRequest): Promise<void>;

  /** Writes a log line for a task run to the engine. */
  putLog(
    taskRunExternalId: string,
    message: string,
    level: LogLevel | undefined,
    retryCount: number,
    extra?: Record<string, unknown>
  ): Promise<void>;

  /** Extends the execution timeout of a task run by `incrementBy` (Go duration string). */
  refreshTimeout(taskRunExternalId: string, incrementBy: string): Promise<void>;

  /** Releases the worker slot held by a task run. */
  releaseSlot(taskRunExternalId: string): Promise<void>;

  /** Streams a chunk of data from a task run. */
  putStream(taskRunExternalId: string, data: string | Uint8Array, index: number): Promise<void>;

  /** Spawns one run. */
  runWorkflow<Q = object, P = object>(
    workflowName: string,
    input: Q,
    options?: SpawnRunOptions
  ): Promise<WorkflowRunRef<P>>;

  /** Spawns many runs in one call. */
  runWorkflows<Q = object, P = object>(runs: SpawnRunRequest<Q>[]): Promise<WorkflowRunRef<P>[]>;

  /** The id the engine assigned to the worker, once registered. */
  workerId(): string | undefined;

  /** Whether the worker serves the given workflow (used by sticky child runs). */
  hasWorkflow(workflowName: string): boolean;

  /** The worker's current labels. */
  workerLabels(): WorkerLabels;

  /** Replaces the worker's labels. */
  upsertWorkerLabels(labels: WorkerLabels): Promise<WorkerLabels>;
}

/**
 * The seam a `DurableContext` talks to for durable events: the worker implements it with
 * `DurableListenerClient` over gRPC, a serverless handler with frames over a websocket.
 */
export interface DurableTransport {
  sendEvent(
    durableTaskExternalId: string,
    invocationCount: number,
    event: RunChildrenEvent
  ): Promise<DurableTaskEventRunAck>;
  sendEvent(
    durableTaskExternalId: string,
    invocationCount: number,
    event: WaitForEvent
  ): Promise<DurableTaskEventWaitForAck>;
  sendEvent(
    durableTaskExternalId: string,
    invocationCount: number,
    event: MemoEvent
  ): Promise<DurableTaskEventMemoAck>;

  waitForCallback(
    durableTaskExternalId: string,
    invocationCount: number,
    branchId: number,
    nodeId: number,
    opts?: { signal?: AbortSignal }
  ): Promise<DurableTaskEventLogEntryResult>;

  consumeCallbackWithoutBlocking(
    durableTaskExternalId: string,
    invocationCount: number,
    branchId: number,
    nodeId: number
  ): void;

  sendMemoCompletedNotification(
    durableTaskExternalId: string,
    nodeId: number,
    branchId: number,
    invocationCount: number,
    memoKey: Uint8Array,
    memoResultPayload?: Uint8Array
  ): Promise<void>;

  cleanupTaskState(durableTaskExternalId: string, invocationCount: number): void;

  sendEvictInvocation(
    durableTaskExternalId: string,
    invocationCount: number,
    reason?: string
  ): Promise<void>;
}

/**
 * Options for constructing a `DurableContext` on top of a `ContextRuntime`.
 */
export interface DurableContextOptions {
  /** The engine version the runtime is talking to; decides whether eviction is supported. */
  engineVersion?: string;
}

export function isContextRuntime(value: unknown): value is ContextRuntime {
  if (typeof value !== 'object' || value === null) return false;
  const candidate = value as Partial<ContextRuntime>;
  return typeof candidate.logger === 'function' && typeof candidate.workerId === 'function';
}
