import { ConcurrencyLimitStrategy, RateLimitDuration } from '@hatchet/protoc/v1/workflows';
import { Conditions } from './conditions';
import { Duration } from './client/duration';
import { InputType, OutputType, UnknownInputType } from './types';
import { Context, DurableContext } from './client/worker/context';
import { EvictionPolicy } from './client/worker/eviction/eviction-policy';
import { WorkerLabelComparator } from '../protoc/v1/shared/trigger';

export { ConcurrencyLimitStrategy, WorkerLabelComparator };

/**
 * Options for configuring the concurrency for a task.
 */
export type Concurrency = {
  /**
   * required the CEL expression to use for concurrency
   *
   * @example
   * ```
   * "input.key" // use the value of the key in the input
   * ```
   */
  expression: string;

  /**
   * (optional) the maximum number of concurrent workflow runs
   *
   * default: 1
   */
  maxRuns?: number;

  /**
   * (optional) the strategy to use when the concurrency limit is reached
   *
   * default: CANCEL_IN_PROGRESS
   */
  limitStrategy?: ConcurrencyLimitStrategy;
};

/**
 * @deprecated use Concurrency instead
 */
export type TaskConcurrency = Concurrency;

/**
 * Base type for idempotency configurations.
 */
export type BaseIdempotencyConfig = {
  /**
   * CEL expression to create an idempotency key from input and metadata.
   * @example "input.id" // use the 'id' field from input as the key
   */
  expression: string;
};

/**
 * TTL-based idempotency: prevents duplicate runs within a sliding time window.
 */
export type TTLBasedIdempotencyConfig = BaseIdempotencyConfig & {
  strategy: 'ttl';

  /**
   * How long the idempotency key should live (in milliseconds).
   */
  ttlMs: number;
};

/**
 * Status-based idempotency: keeps the idempotency key alive until the associated run
 * reaches a terminal status. `fallbackTtlMs` caps how long the key can live before it's evicted.
 */
export type StatusBasedIdempotencyConfig = BaseIdempotencyConfig & {
  strategy: 'status';

  /**
   * Fallback time-to-live (in milliseconds): the longest the idempotency key can live
   * before it's evicted, even if the run has not reached a terminal status.
   */
  fallbackTtlMs: number;
};

/**
 * Union of all supported idempotency configurations.
 */
export type IdempotencyConfig = TTLBasedIdempotencyConfig | StatusBasedIdempotencyConfig;

export class NonRetryableError extends Error {
  constructor(message?: string) {
    super(message);
    this.name = 'NonRetryableError';

    Object.setPrototypeOf(this, new.target.prototype);
  }
}

export type TaskFn<
  I extends InputType = UnknownInputType,
  O extends OutputType = void,
  C = Context<I>,
> = (input: I, ctx: C) => O | Promise<O>;

export type DurableTaskFn<
  I extends InputType = UnknownInputType,
  O extends OutputType = void,
> = TaskFn<I, O, DurableContext<I>>;

/**
 * Options for creating a hatchet task which is an atomic unit of work in a workflow.
 * @template I The input type for the task function.
 * @template O The return type of the task function (can be inferred from the return value of fn).
 */
//= TaskFn<I, O>
export type CreateBaseTaskOpts<
  I extends InputType = UnknownInputType,
  O extends OutputType = void,
  C = TaskFn<I, O>,
> = {
  /**
   * The name of the task.
   */
  name: string;

  /**
   * The function to execute when the task runs.
   * @param input The input data for the workflow invocation.
   * @param ctx The execution context for the task.
   * @returns The result of the task execution.
   */
  fn?: C;

  /**
   * @deprecated use executionTimeout instead
   */
  timeout?: Duration;

  /**
   * (optional) execution timeout duration for the task after it starts running
   * go duration format (e.g., "1s", "5m", "1h").
   *
   * default: 60s
   */
  executionTimeout?: Duration;

  /**
   * (optional) schedule timeout for the task (max duration to allow the task to wait in the queue)
   * go duration format (e.g., "1s", "5m", "1h").
   *
   * default: 5m
   */
  scheduleTimeout?: Duration;

  /**
   * (optional) number of retries for the task.
   *
   * default: 0
   */
  retries?: number;

  /**
   * (optional) backoff strategy configuration for retries.
   * - factor: Base of the exponential backoff (base ^ retry count)
   * - maxSeconds: Maximum backoff duration in seconds
   */
  backoff?: {
    factor?: number | undefined;
    maxSeconds?: number | undefined;
  };

  /**
   * (optional) rate limits for the task.
   */
  rateLimits?: {
    units: string | number;
    key?: string;
    staticKey?: string;
    dynamicKey?: string;
    limit?: string | number;
    duration?: RateLimitDuration;
  }[];

  /**
   * (optional) worker labels for task routing and scheduling.
   * Each label can be a simple string/number value or an object with additional configuration:
   * - value: The label value (string or number)
   * - required: Whether the label is required for worker matching
   * - weight: Priority weight for worker selection
   * - comparator: Custom comparison logic for label matching
   */
  desiredWorkerLabels?: Record<
    string,
    {
      value: string | number;
      required?: boolean;
      weight?: number;
      comparator?: WorkerLabelComparator;
    }
  >;

  /**
   * (optional) the concurrency options for the task
   */
  concurrency?: Concurrency | Concurrency[];

  /**
   * (optional) a CEL expression evaluated against the run's input to produce a
   * human-readable display name for this task. Declared in the task definition
   * and evaluated at trigger time. For a single-task workflow the task-level
   * expression names the run and takes precedence over the workflow-level one.
   * Any evaluation error falls back to the generated `<readableId>-<timestamp>`
   * label. A malformed expression is rejected at registration.
   *
   * @example "'enrich-' + input.customerName"
   */
  displayName?: string;

  /**
   * (optional) the number of default worker slots this task consumes.
   *
   * A worker has a fixed number of slots (default 100), and a normal task consumes one. Set slotCost
   * higher for a task that needs more memory or CPU, so a worker runs fewer of them at once. A single
   * worker must have that many free slots to run it. Not available on durable tasks.
   *
   * default: 1
   */
  slotCost?: number;

  /** @internal */
  slotRequests?: Record<string, number>;
};

export type CreateWorkflowTaskOpts<
  I extends InputType = UnknownInputType,
  O extends OutputType = void,
  C extends TaskFn<I, O> | DurableTaskFn<I, O> = TaskFn<I, O>,
> = CreateBaseTaskOpts<I, O, C> & {
  /**
   * Parent tasks that must complete before this task runs.
   * Used to define the directed acyclic graph (DAG) of the workflow.
   */
  parents?: CreateWorkflowTaskOpts<I, any, any>[];

  /**
   * (optional) the conditions to match before the task is queued
   * all provided conditions must be met (AND logic)
   * use Or() to create a condition that waits for any of the provided conditions to be met (OR logic)
   *
   * @example
   * ```
   * waitFor: [{ sleepFor: 5 }, { eventKey: 'user:update' }] // all conditions must be met
   * ```
   * @example
   * ```
   * waitFor: Or({ eventKey: 'user:update' }, { parent: firstTask }) // any of the conditions must be met
   * ```
   * @example
   * ```
   * waitFor: [{ sleepFor: 5 }, Or({ eventKey: 'user:update' }, { eventKey: 'user:delete' })] // sleep or both user:update or user:delete must be met
   * ```
   */
  waitFor?: Conditions | Conditions[];

  /**
   * (optional) cancel the task if the conditions are met
   * all provided conditions must be met (AND logic)
   * use Or() to create a condition that waits for any of the provided conditions to be met (OR logic)
   *
   * @example
   * ```
   * cancelIf: { eventKey: 'user:update' } // cancel the task if the user:update event is received
   * ```
   * @example
   * ```
   * cancelIf: [{ sleepFor: 5 }, Or({ eventKey: 'user:update' }, { eventKey: 'user:delete' })] // cancel the task if the sleep or both user:update or user:delete are met
   */
  cancelIf?: Conditions | Conditions[];

  /**
   * (optional) skip the task if the conditions are met
   * all provided conditions must be met (AND logic)
   * use Or() to create a condition that waits for any of the provided conditions to be met (OR logic)
   *
   * @example
   * ```
   * skipIf: [{ eventKey: 'user:update' }] // skip the task if the user:update event is received
   * ```
   * @example
   * ```
   * skipIf: [{ sleepFor: 5 }, Or({ eventKey: 'user:update' }, { eventKey: 'user:delete' })] // skip the task if the sleep or both user:update or user:delete are met
   * ```
   * @example
   * ```
   * skipIf: [{ parent: firstTask }] // skip the task if the parent task completes
   * ```
   */
  skipIf?: Conditions | Conditions[];
};

/**
 * Options for creating a hatchet durable task which is an atomic unit of work in a workflow.
 * @template I The input type for the task function.
 * @template O The return type of the task function (can be inferred from the return value of fn).
 */
export type CreateWorkflowDurableTaskOpts<
  I extends InputType = UnknownInputType,
  O extends OutputType = void,
  C extends DurableTaskFn<I, O> = DurableTaskFn<I, O>,
> = Omit<CreateWorkflowTaskOpts<I, O, C>, 'slotCost'> & {
  /**
   * Eviction policy for the durable task. Controls TTL-based eviction and capacity-based eviction.
   * Defaults to the built-in eviction policy when omitted or `undefined`.
   */
  evictionPolicy?: EvictionPolicy;
};

/**
 * Options for configuring the onSuccess task that is invoked when a task succeeds.
 * @template I The input type for the task function.
 * @template O The return type of the task function (can be inferred from the return value of fn).
 */
export type CreateOnSuccessTaskOpts<
  I extends InputType = UnknownInputType,
  O extends OutputType = void,
  C extends TaskFn<I, O> = TaskFn<I, O>,
> = Omit<CreateBaseTaskOpts<I, O, C>, 'name'>;

/**
 * Options for configuring the onFailure task that is invoked when a task fails.
 * @template I The input type for the task function.
 * @template O The return type of the task function (can be inferred from the return value of fn).
 */
export type CreateOnFailureTaskOpts<
  I extends InputType = UnknownInputType,
  O extends OutputType = void,
  C extends TaskFn<I, O> = TaskFn<I, O>,
> = Omit<CreateBaseTaskOpts<I, O, C>, 'name'>;
