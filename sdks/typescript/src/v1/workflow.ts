/* eslint-disable no-dupe-class-members */
import WorkflowRunRef from '@hatchet/util/workflow-run-ref';
import { Context, JsonObject } from '@hatchet/step';
import { CronWorkflows, ScheduledWorkflows } from '@hatchet/clients/rest/generated/data-contracts';
import { Workflow as WorkflowV0 } from '@hatchet/workflow';
import { IHatchetClient } from './client/client.interface';
import { CreateTaskOpts } from './task';

const UNBOUND_ERR = new Error('workflow unbound to hatchet client, hint: use client.run instead');

/**
 * Additional metadata that can be attached to a workflow run.
 */
type AdditionalMetadata = Record<string, string>;

/**
 * Options for running a workflow.
 */
export type RunOpts = {
  /**
   * Additional metadata to attach to the workflow run.
   */
  additionalMetadata?: AdditionalMetadata;
};

/**
 * Extracts a property from an object type based on task name, or falls back to inferred type
 */
type TaskOutputType<K, TaskName extends string, InferredType> = TaskName extends keyof K
  ? K[TaskName]
  : InferredType;

/**
 * Options for creating a new workflow.
 */
export type CreateWorkflowOpts = {
  /**
   * The name of the workflow.
   */
  name: WorkflowV0['id'];
  /**
   * Optional description of the workflow.
   */
  description?: WorkflowV0['description'];
  /**
   * Optional version of the workflow.
   */
  version?: WorkflowV0['version'];
  /**
   * Optional sticky strategy for the workflow.
   */
  sticky?: WorkflowV0['sticky'];
  /**
   * Optional schedule timeout for the workflow.
   */
  scheduleTimeout?: WorkflowV0['scheduleTimeout'];
  /**
   * Optional on config for the workflow.
   */
  on?: WorkflowV0['on'];

  concurrency?: WorkflowV0['concurrency'];
};

/**
 * Internal definition of a workflow and its tasks.
 */
type WorkflowDefinition = CreateWorkflowOpts & {
  /**
   * The tasks that make up this workflow.
   */
  tasks: CreateTaskOpts<any, any>[];

  // TODO on failure
};

/**
 * Represents a workflow that can be executed by Hatchet.
 * @template T The input type for the workflow.
 * @template K The return type of the workflow.
 */
export class Workflow<T extends JsonObject, K> {
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
      tasks: [],
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
  enqueue(input: T, options?: RunOpts): WorkflowRunRef<K> {
    if (!this.client) {
      throw UNBOUND_ERR;
    }

    return this.client.v0.admin.runWorkflow(this.definition.name, input, options);
  }

  /**
   * Executes the workflow with the given input and awaits the results.
   * @param input The input data for the workflow.
   * @param options Optional configuration for this workflow run.
   * @returns A promise that resolves with the workflow result.
   * @throws Error if the workflow is not bound to a Hatchet client.
   */
  async run(input: T, options?: RunOpts): Promise<K>;
  async run(input: T[], options?: RunOpts): Promise<K[]>;
  async run(input: T | T[], options?: RunOpts): Promise<K | K[]> {
    if (!this.client) {
      throw UNBOUND_ERR;
    }

    if (Array.isArray(input)) {
      // FIXME use bulk endpoint?
      return Promise.all(input.map((i) => this.run(i, options)));
    }

    const res = this.client.v0.admin.runWorkflow(this.definition.name, input, options);
    return res.result() as Promise<K>;
  }

  /**
   * Schedules a workflow to run at a specific date and time in the future.
   * @param enqueueAt The date when the workflow should be triggered.
   * @param input The input data for the workflow.
   * @param options Optional configuration for this workflow run.
   * @returns A promise that resolves with the scheduled workflow details.
   * @throws Error if the workflow is not bound to a Hatchet client.
   */
  async schedule(enqueueAt: Date, input: T, options?: RunOpts): Promise<ScheduledWorkflows> {
    if (!this.client) {
      throw UNBOUND_ERR;
    }

    const scheduled = this.client.v0.schedule.create(this.definition.name, {
      triggerAt: enqueueAt,
      input,
      additionalMetadata: options?.additionalMetadata,
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
  async delay(duration: number, input: T, options?: RunOpts): Promise<ScheduledWorkflows> {
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
    input: T,
    options?: RunOpts
  ): Promise<CronWorkflows> {
    if (!this.client) {
      throw UNBOUND_ERR;
    }

    const cronDef = this.client.v0.cron.create(this.definition.name, {
      expression,
      input,
      additionalMetadata: options?.additionalMetadata,
      name,
    });

    return cronDef;
  }

  /**
   * Adds a task to the workflow.
   * The return type will be either the property on K that corresponds to the task name,
   * or if there is no matching property, the inferred return type of the function.
   * @template Name The literal string name of the task.
   * @template L The inferred return type of the task function.
   * @param options The task configuration options.
   * @returns The task options that were added.
   */
  task<Name extends string, L>(
    options: Omit<CreateTaskOpts<T, TaskOutputType<K, Name, L>>, 'fn'> & {
      name: Name;
      fn: (
        input: T,
        ctx: Context<T>
      ) => TaskOutputType<K, Name, L> | Promise<TaskOutputType<K, Name, L>>;
    }
  ): CreateTaskOpts<T, TaskOutputType<K, Name, L>> {
    const typedOptions = options as CreateTaskOpts<T, TaskOutputType<K, Name, L>>;
    this.definition.tasks.push(typedOptions);
    return typedOptions;
  }

  // @deprecated use definition.name instead
  get id() {
    return this.definition.name;
  }
}

/**
 * Creates a new workflow instance.
 * @template T The input type for the workflow.
 * @template K The return type of the workflow.
 * @param options The options for creating the workflow.
 * @param client Optional Hatchet client instance.
 * @returns A new Workflow instance.
 */
export function CreateWorkflow<T extends JsonObject = any, K = any>(
  options: CreateWorkflowOpts,
  client?: IHatchetClient
): Workflow<T, K> {
  return new Workflow<T, K>(options, client);
}
