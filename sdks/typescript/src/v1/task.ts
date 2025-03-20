import { ConcurrencyLimitStrategy } from '@hatchet/protoc/v1/workflows';
import { Context, CreateStep } from '@hatchet/step';
import { Conditions } from './conditions';

/**
 * Options for configuring the concurrency for a task.
 */
export type TaskConcurrency = {
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

export type TaskFn<T, K> = (input: T, ctx: Context<T>) => K | Promise<K>;

/**
 * Options for creating a hatchet task which is an atomic unit of work in a workflow.
 * @template T The input type for the task function.
 * @template K The return type of the task function (can be inferred from the return value of fn).
 */
export type CreateTaskOpts<T, K> = {
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
  fn: TaskFn<T, K>;

  /**
   * Parent tasks that must complete before this task runs.
   * Used to define the directed acyclic graph (DAG) of the workflow.
   */
  parents?: CreateTaskOpts<T, any>[];

  /**
   * @deprecated use executionTimeout instead
   */
  timeout?: CreateStep<T, K>['timeout'];

  /**
   * (optional) execution timeout duration for the task after it starts running
   * go duration format (e.g., "1s", "5m", "1h").
   *
   * default: 60s
   */
  executionTimeout?: CreateStep<T, K>['timeout'];

  /**
   * (optional) schedule timeout for the task (max duration to allow the task to wait in the queue)
   * go duration format (e.g., "1s", "5m", "1h").
   *
   * default: 5m
   */
  scheduleTimeout?: CreateStep<T, K>['timeout'];

  /**
   * (optional) number of retries for the task.
   *
   * default: 0
   */
  retries?: CreateStep<T, K>['retries'];

  /**
   * (optional) backoff strategy configuration for retries.
   * - factor: Base of the exponential backoff (base ^ retry count)
   * - maxSeconds: Maximum backoff duration in seconds
   */
  backoff?: CreateStep<T, K>['backoff'];

  /**
   * (optional) rate limits for the task.
   */
  rateLimits?: CreateStep<T, K>['rate_limits'];

  /**
   * (optional) worker labels for task routing and scheduling.
   * Each label can be a simple string/number value or an object with additional configuration:
   * - value: The label value (string or number)
   * - required: Whether the label is required for worker matching
   * - weight: Priority weight for worker selection
   * - comparator: Custom comparison logic for label matching
   */
  workerLabels?: CreateStep<T, K>['worker_labels'];

  /**
   * (optional) the concurrency options for the task
   */
  concurrency?: TaskConcurrency | TaskConcurrency[];

  /**
   * (optional) the conditions to match before the task is queued
   */
  queueIf?: Conditions | Conditions[];

  /**
   * (optional) the conditions to match before the task is queued
   */
  cancelIf?: Conditions | Conditions[];

  /**
   * (optional) the conditions to match before the task is skipped
   */
  skipIf?: Conditions | Conditions[];
};

/**
 * Options for configuring the onFailure task that is invoked when a task fails.
 * @template T The input type for the task function.
 * @template K The return type of the task function (can be inferred from the return value of fn).
 */
export type CreateOnFailureTaskOpts<T, K> = {
  /**
   * The function to execute when the task runs.
   * @param input The input data for the workflow invocation.
   * @param ctx The execution context for the task.
   * @returns The result of the task execution.
   */
  fn: TaskFn<T, K>;

  /**
   * (optional) execution timeout duration for the onFailure task after it starts running
   * go duration format (e.g., "1s", "5m", "1h").
   *
   * default: 60s
   */
  executionTimeout?: CreateStep<T, K>['timeout'];

  /**
   * (optional) schedule timeout for the onFailure task (max duration to allow the task to wait in the queue)
   * go duration format (e.g., "1s", "5m", "1h").
   *
   * default: 5m
   */
  scheduleTimeout?: CreateStep<T, K>['timeout'];

  /**
   * (optional) number of retries for the onFailure task.
   *
   * default: 0
   */
  retries?: CreateStep<T, K>['retries'];

  /**
   * (optional) backoff strategy configuration for retries.
   * - factor: Base of the exponential backoff (base ^ retry count)
   * - maxSeconds: Maximum backoff duration in seconds
   */
  backoff?: CreateStep<T, K>['backoff'];

  /**
   * (optional) rate limits for the onFailure task.
   */
  rateLimits?: CreateStep<T, K>['rate_limits'];

  /**
   * (optional) worker labels for task routing and scheduling.
   * Each label can be a simple string/number value or an object with additional configuration:
   * - value: The label value (string or number)
   * - required: Whether the label is required for worker matching
   * - weight: Priority weight for worker selection
   * - comparator: Custom comparison logic for label matching
   */
  workerLabels?: CreateStep<T, K>['worker_labels'];
};
