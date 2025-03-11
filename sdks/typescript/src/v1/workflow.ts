import WorkflowRunRef from '@hatchet/util/workflow-run-ref';
import { Context } from '@hatchet/step';
import { IHatchetClient } from './client/client.interface';
import { CreateTaskOpts } from './task';

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
 * Internal definition of a workflow and its tasks.
 */
type WorkflowDefinition = {
  /**
   * The name of the workflow.
   */
  name: string;
  /**
   * The tasks that make up this workflow.
   */
  tasks: CreateTaskOpts<any, any>[];
};

/**
 * Options for creating a new workflow.
 */
export type CreateWorkflowOpts = {
  /**
   * The name of the workflow.
   */
  name: string;
  /**
   * Optional description of the workflow.
   */
  description?: string;
};

/**
 * Represents a workflow that can be executed by Hatchet.
 * @template T The input type for the workflow.
 * @template K The return type of the workflow.
 */
export class Workflow<T, K> {
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
      name: options.name,
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
  enqueue(input: T, options: RunOpts): WorkflowRunRef<K> {
    if (!this.client) {
      throw new Error('workflow unbound to hatchet client, hint: use client.run instead');
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
  async run(input: T, options?: RunOpts): Promise<K> {
    if (!this.client) {
      throw new Error('workflow unbound to hatchet client, hint: use client.run instead');
    }

    const res = this.client.v0.admin.runWorkflow(this.definition.name, input, options);
    return res.result() as Promise<K>;
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
  addTask<Name extends string, L>(
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
}

/**
 * Creates a new workflow instance.
 * @template T The input type for the workflow.
 * @template K The return type of the workflow.
 * @param options The options for creating the workflow.
 * @param client Optional Hatchet client instance.
 * @returns A new Workflow instance.
 */
export function CreateWorkflow<T = any, K = any>(
  options: CreateWorkflowOpts,
  client?: IHatchetClient
): Workflow<T, K> {
  return new Workflow<T, K>(options, client);
}
