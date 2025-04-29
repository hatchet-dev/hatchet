/* eslint-disable max-classes-per-file */
/* eslint-disable no-underscore-dangle */
/* eslint-disable no-dupe-class-members */
import WorkflowRunRef from '@hatchet/util/workflow-run-ref';
import { CronWorkflows, ScheduledWorkflows } from '@hatchet/clients/rest/generated/data-contracts';
import { Workflow as WorkflowV0 } from '@hatchet/workflow';
import { IHatchetClient } from './client/client.interface';
import {
  CreateWorkflowTaskOpts,
  CreateOnFailureTaskOpts,
  TaskFn,
  CreateWorkflowDurableTaskOpts,
  CreateBaseTaskOpts,
  CreateOnSuccessTaskOpts,
  Concurrency,
  DurableTaskFn,
} from './task';
import { Duration } from './client/duration';
import { MetricsClient } from './client/features/metrics';
import { InputType, OutputType, UnknownInputType, JsonObject } from './types';
import { Context, DurableContext } from './client/worker/context';

const UNBOUND_ERR = new Error('workflow unbound to hatchet client, hint: use client.run instead');

// eslint-disable-next-line no-shadow
export enum Priority {
  LOW = 1,
  MEDIUM = 2,
  HIGH = 3,
}

/**
 * Additional metadata that can be attached to a workflow run.
 */
type AdditionalMetadata = Record<string, string>;

/**
 * Options for running a workflow.
 */
export type RunOpts = {
  /**
   * (optional) additional metadata to attach to the workflow run.
   */
  additionalMetadata?: AdditionalMetadata;

  /**
   * (optional) the priority for the workflow run.
   *
   * values: Priority.LOW, Priority.MEDIUM, Priority.HIGH (1, 2, or 3 )
   */
  priority?: Priority;
};

/**
 * Helper type to safely extract output types from task results
 */
export type TaskOutput<O, Key extends string, Fallback> =
  O extends Record<Key, infer Value> ? (Value extends OutputType ? Value : Fallback) : Fallback;

/**
 * Extracts a property from an object type based on task name, or falls back to inferred type
 */
export type TaskOutputType<
  O,
  TaskName extends string,
  InferredType extends OutputType,
> = TaskName extends keyof O
  ? O[TaskName] extends OutputType
    ? O[TaskName]
    : InferredType
  : InferredType;

export type CreateBaseWorkflowOpts = {
  /**
   * The name of the workflow.
   */
  name: WorkflowV0['id'];
  /**
   * (optional) description of the workflow.
   */
  description?: WorkflowV0['description'];
  /**
   * (optional) version of the workflow.
   */
  version?: WorkflowV0['version'];
  /**
   * (optional) sticky strategy for the workflow.
   */
  sticky?: WorkflowV0['sticky'];

  /**
   * (optional) on config for the workflow.
   * @deprecated use onCrons and onEvents instead
   */
  on?: WorkflowV0['on'];

  /**
   * (optional) cron config for the workflow.
   */
  onCrons?: string[];

  /**
   * (optional) event config for the workflow.
   */
  onEvents?: string[];

  /**
   * (optional) concurrency config for the workflow.
   */
  concurrency?: Concurrency | Concurrency[];

  /**
   * (optional) the priority for the workflow.
   * values: Priority.LOW, Priority.MEDIUM, Priority.HIGH (1, 2, or 3 )
   */
  defaultPriority?: Priority;
};

export type CreateTaskWorkflowOpts<
  I extends InputType = UnknownInputType,
  O extends OutputType = void,
> = CreateBaseWorkflowOpts & CreateBaseTaskOpts<I, O, TaskFn<I, O>>;

export type CreateDurableTaskWorkflowOpts<
  I extends InputType = UnknownInputType,
  O extends OutputType = void,
> = CreateBaseWorkflowOpts & CreateBaseTaskOpts<I, O, DurableTaskFn<I, O>>;

/**
 * Options for creating a new workflow.
 */
export type CreateWorkflowOpts = CreateBaseWorkflowOpts & {
  /**
   * (optional) default configuration for all tasks in the workflow.
   */
  taskDefaults?: TaskDefaults;
};

/**
 * Default configuration for all tasks in the workflow.
 * Can be overridden by task-specific options.
 */
export type TaskDefaults = {
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
  retries?: CreateWorkflowTaskOpts<any, any>['retries'];

  /**
   * (optional) backoff strategy configuration for retries.
   * - factor: Base of the exponential backoff (base ^ retry count)
   * - maxSeconds: Maximum backoff duration in seconds
   */
  backoff?: CreateWorkflowTaskOpts<any, any>['backoff'];

  /**
   * (optional) rate limits for the task.
   */
  rateLimits?: CreateWorkflowTaskOpts<any, any>['rateLimits'];

  /**
   * (optional) worker labels for task routing and scheduling.
   * Each label can be a simple string/number value or an object with additional configuration:
   * - value: The label value (string or number)
   * - required: Whether the label is required for worker matching
   * - weight: Priority weight for worker selection
   * - comparator: Custom comparison logic for label matching
   */
  workerLabels?: CreateWorkflowTaskOpts<any, any>['desiredWorkerLabels'];

  /**
   * (optional) the concurrency options for the task.
   */
  concurrency?: Concurrency | Concurrency[];
};

/**
 * Internal definition of a workflow and its tasks.
 */
export type WorkflowDefinition = CreateWorkflowOpts & {
  /**
   * The tasks that make up this workflow.
   */
  _tasks: CreateWorkflowTaskOpts<any, any>[];

  /**
   * The durable tasks that make up this workflow.
   */
  _durableTasks: CreateWorkflowDurableTaskOpts<any, any>[];

  /**
   * (optional) onFailure handler for the workflow.
   * Invoked when any task in the workflow fails.
   * @param ctx The context of the workflow.
   */
  onFailure?: TaskFn<any, any> | CreateOnFailureTaskOpts<any, any>;

  /**
   * (optional) onSuccess handler for the workflow.
   * Invoked when all tasks in the workflow complete successfully.
   * @param ctx The context of the workflow.
   */
  onSuccess?: TaskFn<any, any> | CreateOnSuccessTaskOpts<any, any>;
};

/**
 * Represents a workflow that can be executed by Hatchet.
 * @template I The input type for the workflow.
 * @template O The return type of the workflow.
 */
export class BaseWorkflowDeclaration<
  I extends InputType = UnknownInputType,
  O extends OutputType = void,
> {
  /**
   * The Hatchet client instance used to execute the workflow.
   */
  client: IHatchetClient | undefined;

  /**
   * The internal workflow definition.
   */
  definition: WorkflowDefinition;

  /**
   * Creates a new workflow instance.
   * @param options The options for creating the workflow.
   * @param client Optional Hatchet client instance.
   */
  constructor(options: CreateWorkflowOpts, client?: IHatchetClient) {
    this.definition = {
      ...options,
      _tasks: [],
      _durableTasks: [],
    };

    this.client = client;
  }

  /**
   * Triggers a workflow run without waiting for completion.
   * @param input The input data for the workflow.
   * @param options Optional configuration for this workflow run.
   * @returns A WorkflowRunRef containing the run ID and methods to get results and interact with the run.
   * @throws Error if the workflow is not bound to a Hatchet client.
   */
  async runNoWait(
    input: I,
    options?: RunOpts,
    _standaloneTaskName?: string
  ): Promise<WorkflowRunRef<O>> {
    if (!this.client) {
      throw UNBOUND_ERR;
    }

    const res = await this.client.admin.runWorkflow<I, O>(this.name, input, options);

    if (_standaloneTaskName) {
      res._standalone_task_name = _standaloneTaskName;
    }

    return res;
  }

  /**
   * @alias run
   * Triggers a workflow run and waits for the result.
   * @template I - The input type for the workflow
   * @template O - The return type of the workflow
   * @param input - The input data for the workflow
   * @param options - Configuration options for the workflow run
   * @returns A promise that resolves with the workflow result
   */
  async runAndWait(input: I, options?: RunOpts, _standaloneTaskName?: string): Promise<O>;
  async runAndWait(input: I[], options?: RunOpts, _standaloneTaskName?: string): Promise<O[]>;
  async runAndWait(
    input: I | I[],
    options?: RunOpts,
    _standaloneTaskName?: string
  ): Promise<O | O[]> {
    if (!this.client) {
      throw UNBOUND_ERR;
    }

    if (Array.isArray(input)) {
      return Promise.all(input.map((i) => this.runAndWait(i, options, _standaloneTaskName)));
    }

    return this.run(input, options, _standaloneTaskName);
  }
  /**
   * Executes the workflow with the given input and awaits the results.
   * @param input The input data for the workflow.
   * @param options Optional configuration for this workflow run.
   * @returns A promise that resolves with the workflow result.
   * @throws Error if the workflow is not bound to a Hatchet client.
   */
  async run(input: I, options?: RunOpts, _standaloneTaskName?: string): Promise<O>;
  async run(input: I[], options?: RunOpts, _standaloneTaskName?: string): Promise<O[]>;
  async run(input: I | I[], options?: RunOpts, _standaloneTaskName?: string): Promise<O | O[]> {
    if (!this.client) {
      throw UNBOUND_ERR;
    }

    if (Array.isArray(input)) {
      let resp: WorkflowRunRef<O>[] = [];
      for (let i = 0; i < input.length; i += 500) {
        const batch = input.slice(i, i + 500);
        const batchResp = await this.client.admin.runWorkflows<I, O>(
          batch.map((inp) => ({
            workflowName: this.definition.name,
            input: inp,
            options,
          }))
        );
        resp = resp.concat(batchResp);
      }

      const res: Promise<O>[] = [];
      resp.forEach((ref, index) => {
        const wf = input[index].workflow;
        if (wf instanceof TaskWorkflowDeclaration) {
          // eslint-disable-next-line no-param-reassign
          ref._standalone_task_name = wf._standalone_task_name;
        }
        res.push(ref.result());
      });
      return Promise.all(res);
    }

    const res = await this.client.admin.runWorkflow<I, O>(this.definition.name, input, options);

    if (_standaloneTaskName) {
      res._standalone_task_name = _standaloneTaskName;
    }

    return res.result() as Promise<O>;
  }

  /**
   * Schedules a workflow to run at a specific date and time in the future.
   * @param enqueueAt The date when the workflow should be triggered.
   * @param input The input data for the workflow.
   * @param options Optional configuration for this workflow run.
   * @returns A promise that resolves with the scheduled workflow details.
   * @throws Error if the workflow is not bound to a Hatchet client.
   */
  async schedule(enqueueAt: Date, input: I, options?: RunOpts): Promise<ScheduledWorkflows> {
    if (!this.client) {
      throw UNBOUND_ERR;
    }

    const scheduled = this.client._v0.schedule.create(this.definition.name, {
      triggerAt: enqueueAt,
      input: input as JsonObject,
      ...options,
    });

    return scheduled;
  }

  /**
   * Schedules a workflow to run after a specified delay.
   * @param duration The delay in seconds before the workflow should run.
   * @param input The input data for the workflow.
   * @param options Optional configuration for this workflow run.
   * @returns A promise that resolves with the scheduled workflow details.
   * @throws Error if the workflow is not bound to a Hatchet client.
   */
  async delay(duration: number, input: I, options?: RunOpts): Promise<ScheduledWorkflows> {
    const now = Date.now();
    const triggerAt = new Date(now + duration * 1000);
    return this.schedule(triggerAt, input, options);
  }

  /**
   * Creates a cron schedule for the workflow.
   * @param name The name of the cron schedule.
   * @param expression The cron expression defining the schedule.
   * @param input The input data for the workflow.
   * @param options Optional configuration for this workflow run.
   * @returns A promise that resolves with the cron workflow details.
   * @throws Error if the workflow is not bound to a Hatchet client.
   */
  async cron(
    name: string,
    expression: string,
    input: I,
    options?: RunOpts
  ): Promise<CronWorkflows> {
    if (!this.client) {
      throw UNBOUND_ERR;
    }

    const cronDef = this.client._v0.cron.create(this.definition.name, {
      expression,
      input: input as JsonObject,
      ...options,
      additionalMetadata: options?.additionalMetadata,
      name,
    });

    return cronDef;
  }

  /**
   * Get metrics for the workflow.
   * @param opts Optional configuration for the metrics request.
   * @returns A promise that resolves with the workflow metrics.
   * @throws Error if the workflow is not bound to a Hatchet client.
   */
  metrics(opts?: Parameters<MetricsClient['getWorkflowMetrics']>[1]) {
    if (!this.client) {
      throw UNBOUND_ERR;
    }

    return this.client.metrics.getWorkflowMetrics(this.definition.name, opts);
  }

  /**
   * Get queue metrics for the workflow.
   * @param opts Optional configuration for the metrics request.
   * @returns A promise that resolves with the workflow metrics.
   * @throws Error if the workflow is not bound to a Hatchet client.
   */
  queueMetrics(opts?: Omit<Parameters<MetricsClient['getQueueMetrics']>[0], 'workflows'>) {
    if (!this.client) {
      throw UNBOUND_ERR;
    }

    return this.client.metrics.getQueueMetrics({
      ...opts,
      workflows: [this.definition.name],
    });
  }

  /**
   * Get the current state of the workflow.
   * @returns A promise that resolves with the workflow state.
   * @throws Error if the workflow is not bound to a Hatchet client.
   */
  get() {
    if (!this.client) {
      throw UNBOUND_ERR;
    }

    return this.client.workflows.get(this);
  }

  // // gets the pause state of the workflow
  // isPaused() {
  //   if (!this.client) {
  //     throw UNBOUND_ERR;
  //   }

  //   return this.client.workflows.isPaused(this);
  // }

  // // pause assignment of workflow
  // pause() {
  //   if (!this.client) {
  //     throw UNBOUND_ERR;
  //   }

  //   return this.client.workflows.pause(this);
  // }

  // // unpause assignment of workflow
  // unpause() {
  //   if (!this.client) {
  //     throw UNBOUND_ERR;
  //   }

  //   return this.client.workflows.unpause(this);
  // }

  // @deprecated use definition.name instead
  get id() {
    return this.definition.name;
  }

  /**
   * Get the friendly name of the workflow.
   * @returns The name of the workflow.
   */
  get name() {
    return this.definition.name;
  }
}

export class WorkflowDeclaration<
  I extends InputType = UnknownInputType,
  O extends OutputType = void,
> extends BaseWorkflowDeclaration<I, O> {
  /**
   * Adds a task to the workflow.
   * The return type will be either the property on O that corresponds to the task name,
   * or if there is no matching property, the inferred return type of the function.
   * @template Name The literal string name of the task.
   * @template Fn The type of the task function.
   * @param options The task configuration options.
   * @returns The task options that were added.
   */
  task<
    Name extends string,
    Fn extends Name extends keyof O
      ? (
          input: I,
          ctx: Context<I>
        ) => O[Name] extends OutputType ? O[Name] | Promise<O[Name]> : void
      : (input: I, ctx: Context<I>) => void,
    FnReturn = ReturnType<Fn> extends Promise<infer P> ? P : ReturnType<Fn>,
    TO extends OutputType = Name extends keyof O
      ? O[Name] extends OutputType
        ? O[Name]
        : never
      : FnReturn extends OutputType
        ? FnReturn
        : never,
  >(
    options:
      | (Omit<CreateWorkflowTaskOpts<I, TO>, 'fn'> & {
          name: Name;
          fn: Fn;
        })
      | TaskWorkflowDeclaration<I, TO>
  ): CreateWorkflowTaskOpts<I, TO> {
    let typedOptions: CreateWorkflowTaskOpts<I, TO>;

    if (options instanceof TaskWorkflowDeclaration) {
      typedOptions = options.taskDef;
    } else {
      typedOptions = options as CreateWorkflowTaskOpts<I, TO>;
    }

    this.definition._tasks.push(typedOptions);
    return typedOptions;
  }

  /**
   * Adds an onFailure task to the workflow.
   * This will only run if any task in the workflow fails.
   * @template Name The literal string name of the task.
   * @template L The inferred return type of the task function.
   * @param options The task configuration options.
   * @returns The task options that were added.
   */
  onFailure<Name extends string, L extends OutputType>(
    options:
      | (Omit<CreateOnFailureTaskOpts<I, TaskOutputType<O, Name, L>>, 'fn'> & {
          fn: (
            input: I,
            ctx: Context<I>
          ) => TaskOutputType<O, Name, L> | Promise<TaskOutputType<O, Name, L>>;
        })
      | TaskWorkflowDeclaration<any, any>
  ): CreateWorkflowTaskOpts<I, TaskOutputType<O, Name, L>> {
    let typedOptions: CreateWorkflowTaskOpts<I, TaskOutputType<O, Name, L>>;

    if (options instanceof TaskWorkflowDeclaration) {
      typedOptions = options.taskDef;
    } else {
      typedOptions = options as CreateWorkflowTaskOpts<I, TaskOutputType<O, Name, L>>;
    }

    if (this.definition.onFailure) {
      this.client?._v0.logger.warn(`onFailure task will override existing onFailure task`);
    }

    this.definition.onFailure = typedOptions;
    return typedOptions;
  }

  /**
   * Adds an onSuccess task to the workflow.
   * This will only run if all tasks in the workflow complete successfully.
   * @template Name The literal string name of the task.
   * @template L The inferred return type of the task function.
   * @param options The task configuration options.
   * @returns The task options that were added.
   */
  onSuccess<Name extends string, L extends OutputType>(
    options:
      | (Omit<CreateOnSuccessTaskOpts<I, TaskOutputType<O, Name, L>>, 'fn'> & {
          fn: (
            input: I,
            ctx: Context<I>
          ) => TaskOutputType<O, Name, L> | Promise<TaskOutputType<O, Name, L>>;
        })
      | TaskWorkflowDeclaration<any, any>
  ): CreateWorkflowTaskOpts<I, TaskOutputType<O, Name, L>> {
    let typedOptions: CreateWorkflowTaskOpts<I, TaskOutputType<O, Name, L>>;

    if (options instanceof TaskWorkflowDeclaration) {
      typedOptions = options.taskDef;
    } else {
      typedOptions = options as CreateWorkflowTaskOpts<I, TaskOutputType<O, Name, L>>;
    }

    if (this.definition.onSuccess) {
      this.client?._v0.logger.warn(`onSuccess task will override existing onSuccess task`);
    }

    this.definition.onSuccess = typedOptions;
    return typedOptions;
  }

  /**
   * Adds a durable task to the workflow.
   * The return type will be either the property on O that corresponds to the task name,
   * or if there is no matching property, the inferred return type of the function.
   * @template Name The literal string name of the task.
   * @template Fn The type of the task function.
   * @param options The task configuration options.
   * @returns The task options that were added.
   */
  durableTask<
    Name extends string,
    Fn extends Name extends keyof O
      ? (
          input: I,
          ctx: DurableContext<I>
        ) => O[Name] extends OutputType ? O[Name] | Promise<O[Name]> : void
      : (input: I, ctx: DurableContext<I>) => void,
    FnReturn = ReturnType<Fn> extends Promise<infer P> ? P : ReturnType<Fn>,
    TO extends OutputType = Name extends keyof O
      ? O[Name] extends OutputType
        ? O[Name]
        : never
      : FnReturn extends OutputType
        ? FnReturn
        : never,
  >(
    options: Omit<CreateWorkflowTaskOpts<I, TO>, 'fn'> & {
      name: Name;
      fn: Fn;
    }
  ): CreateWorkflowDurableTaskOpts<I, TO> {
    const typedOptions = options as unknown as CreateWorkflowDurableTaskOpts<I, TO>;
    this.definition._durableTasks.push(typedOptions);
    return typedOptions;
  }
}

export class TaskWorkflowDeclaration<
  I extends InputType = UnknownInputType,
  O extends OutputType = void,
> extends BaseWorkflowDeclaration<I, O> {
  _standalone_task_name: string;

  constructor(options: CreateTaskWorkflowOpts<I, O>, client?: IHatchetClient) {
    super({ ...options }, client);

    this._standalone_task_name = options.name;

    this.definition._tasks.push({
      ...options,
    });
  }

  async run(input: I, options?: RunOpts): Promise<O>;
  async run(input: I[], options?: RunOpts): Promise<O[]>;
  async run(input: I | I[], options?: RunOpts): Promise<O | O[]> {
    return (await super.run(input as I, options, this._standalone_task_name)) as O | O[];
  }

  /**
   * Triggers a workflow run without waiting for completion.
   * @param input The input data for the workflow.
   * @param options Optional configuration for this workflow run.
   * @returns A WorkflowRunRef containing the run ID and methods to get results and interact with the run.
   * @throws Error if the workflow is not bound to a Hatchet client.
   */
  async runNoWait(input: I, options?: RunOpts): Promise<WorkflowRunRef<O>> {
    if (!this.client) {
      throw UNBOUND_ERR;
    }

    return super.runNoWait(input, options, this._standalone_task_name);
  }

  get taskDef() {
    return this.definition._tasks[0];
  }
}

/**
 * Creates a new task workflow declaration with types inferred from the function parameter.
 * @template Fn The type of the task function
 * @param options The task configuration options.
 * @param client Optional Hatchet client instance.
 * @returns A new TaskWorkflowDeclaration with inferred types.
 */
export function CreateTaskWorkflow<
  Fn extends (input: I, ctx?: any) => O | Promise<O>,
  I extends InputType = Parameters<Fn>[0],
  O extends OutputType = ReturnType<Fn> extends Promise<infer P>
    ? P extends OutputType
      ? P
      : void
    : ReturnType<Fn> extends OutputType
      ? ReturnType<Fn>
      : void,
>(
  options: {
    fn: Fn;
  } & Omit<CreateTaskWorkflowOpts<I, O>, 'fn'>,
  client?: IHatchetClient
): TaskWorkflowDeclaration<I, O> {
  return new TaskWorkflowDeclaration<I, O>(options as any, client);
}

/**
 * Creates a new workflow instance.
 * @template I The input type for the workflow.
 * @template O The return type of the workflow.
 * @param options The options for creating the workflow.
 * @param client Optional Hatchet client instance.
 * @returns A new Workflow instance.
 */
export function CreateWorkflow<I extends InputType = UnknownInputType, O extends OutputType = void>(
  options: CreateWorkflowOpts,
  client?: IHatchetClient
): WorkflowDeclaration<I, O> {
  return new WorkflowDeclaration<I, O>(options, client);
}

/**
 * Creates a new durable task workflow declaration with types inferred from the function parameter.
 * @template Fn The type of the durable task function
 * @param options The durable task configuration options.
 * @param client Optional Hatchet client instance.
 * @returns A new TaskWorkflowDeclaration with inferred types.
 */
export function CreateDurableTaskWorkflow<
  // Extract input and return types from the function, but ensure they extend JsonObject
  Fn extends (input: I, ctx: DurableContext<I>) => O | Promise<O>,
  I extends JsonObject = Parameters<Fn>[0],
  O extends JsonObject = ReturnType<Fn> extends Promise<infer P>
    ? P extends JsonObject
      ? P
      : never
    : ReturnType<Fn> extends JsonObject
      ? ReturnType<Fn>
      : never,
>(
  options: {
    fn: Fn;
  } & Omit<CreateWorkflowDurableTaskOpts<I, O>, 'fn'>,
  client?: IHatchetClient
): TaskWorkflowDeclaration<I, O> {
  // Note: We're using TaskWorkflowDeclaration here since task and durableTask
  // share the same declaration structure but with different task types
  const taskWorkflow = new TaskWorkflowDeclaration<I, O>(options as any, client);

  // Move the task from tasks to durableTasks
  if (taskWorkflow.definition._tasks.length > 0) {
    const task = taskWorkflow.definition._tasks[0];
    taskWorkflow.definition._tasks = [];
    taskWorkflow.definition._durableTasks.push(task);
  }

  return taskWorkflow;
}
