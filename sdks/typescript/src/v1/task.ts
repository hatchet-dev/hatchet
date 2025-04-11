import { ConcurrencyLimitStrategy } from '@hatchet/protoc/v1/workflows';
import { Context, CreateStep, DurableContext } from '@hatchet/step';
import { Conditions } from './conditions';
import { Duration } from './client/duration';
import { InputType, OutputType, UnknownInputType } from './types';

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
  fn: C;

  /**
   * @deprecated use executionTimeout instead
   */
  timeout?: CreateStep<I, O>['timeout'];

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
  retries?: CreateStep<I, O>['retries'];

  /**
   * (optional) backoff strategy configuration for retries.
   * - factor: Base of the exponential backoff (base ^ retry count)
   * - maxSeconds: Maximum backoff duration in seconds
   */
  backoff?: CreateStep<I, O>['backoff'];

  /**
   * (optional) rate limits for the task.
   */
  rateLimits?: CreateStep<I, O>['rate_limits'];

  /**
   * (optional) worker labels for task routing and scheduling.
   * Each label can be a simple string/number value or an object with additional configuration:
   * - value: The label value (string or number)
   * - required: Whether the label is required for worker matching
   * - weight: Priority weight for worker selection
   * - comparator: Custom comparison logic for label matching
   */
  desiredWorkerLabels?: CreateStep<I, O>['worker_labels'];

  /**
   * (optional) the concurrency options for the task
   */
  concurrency?: TaskConcurrency | TaskConcurrency[];
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
> = CreateWorkflowTaskOpts<I, O, C>;

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
