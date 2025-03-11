import { Context, CreateStep } from '@hatchet/step';

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
  fn: (input: T, ctx: Context<T>) => K | Promise<K>;

  /**
   * Parent tasks that must complete before this task runs.
   * Used to define the directed acyclic graph (DAG) of the workflow.
   */
  parents?: CreateTaskOpts<T, any>[];

  /**
   * Timeout duration for the task in Go duration format (e.g., "1s", "5m", "1h").
   */
  timeout?: CreateStep<T, K>['timeout'];

  /**
   * Optional retry configuration for the task.
   */
  retries?: CreateStep<T, K>['retries'];

  /**
   * Backoff strategy configuration for retries.
   * - factor: Multiplier for exponential backoff
   * - maxSeconds: Maximum backoff duration in seconds
   */
  backoff?: CreateStep<T, K>['backoff'];

  /**
   * Optional rate limiting configuration for the task.
   */
  rateLimits?: CreateStep<T, K>['rate_limits'];

  /**
   * Worker labels for task routing and scheduling.
   * Each label can be a simple string/number value or an object with additional configuration:
   * - value: The label value (string or number)
   * - required: Whether the label is required for worker matching
   * - weight: Priority weight for worker selection
   * - comparator: Custom comparison logic for label matching
   */
  workerLabels?: CreateStep<T, K>['worker_labels'];
};
