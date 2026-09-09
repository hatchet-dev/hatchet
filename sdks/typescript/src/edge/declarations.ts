import {
  CreateBatchTaskWorkflow,
  CreateDurableTaskWorkflow,
  CreateTaskWorkflow,
  CreateWorkflow,
  CreateBatchTaskWorkflowOpts,
  CreateDurableTaskWorkflowOpts,
  CreateTaskWorkflowOpts,
  CreateWorkflowOpts,
  TaskWorkflowDeclaration,
  WorkflowDeclaration,
} from '@hatchet/v1/declaration';
import type {
  InputType,
  JsonObject,
  OutputType,
  StrictWorkflowOutputType,
  UnknownInputType,
} from '@hatchet/v1/types';
import type { BatchTaskConfig, BatchTaskFn } from '@hatchet/v1/task';
import type { DurableContext } from '@hatchet/v1/client/worker/context';

/**
 * The declaration factories of `HatchetClient` (`task`, `durableTask`, `workflow`,
 * `batchTask`) bound to no client. The declarations they return can be registered and
 * served but cannot be run, scheduled or triggered from the process that declared them.
 */
export interface Declarations {
  /**
   * Declares a task. Types come from generics or are inferred from `fn`.
   */
  task<I extends InputType = UnknownInputType, O extends OutputType = void>(
    options: CreateTaskWorkflowOpts<I, O>
  ): TaskWorkflowDeclaration<I, O>;
  task<
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
    options: { fn: Fn } & Omit<CreateTaskWorkflowOpts<I, O>, 'fn'>
  ): TaskWorkflowDeclaration<I, O>;

  /**
   * Declares a durable task. Types come from generics or are inferred from `fn`.
   */
  durableTask<I extends InputType, O extends OutputType>(
    options: CreateDurableTaskWorkflowOpts<I, O>
  ): TaskWorkflowDeclaration<I, O>;
  durableTask<
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
    options: { fn: Fn } & Omit<CreateDurableTaskWorkflowOpts<I, O>, 'fn'>
  ): TaskWorkflowDeclaration<I, O>;

  /**
   * Declares a workflow: a DAG of tasks added with `workflow.task(...)`.
   */
  workflow<I extends InputType = UnknownInputType, O extends StrictWorkflowOutputType = {}>(
    options: CreateWorkflowOpts
  ): WorkflowDeclaration<I, O>;

  /**
   * Declares a batch task. Types come from generics or are inferred from `fn`.
   *
   * Preview: batch tasks are in beta and may change in future releases.
   */
  batchTask<I extends InputType = UnknownInputType, O extends OutputType = void>(
    options: CreateBatchTaskWorkflowOpts<I, O>
  ): TaskWorkflowDeclaration<I, O>;
  batchTask<
    Fn extends BatchTaskFn<I, O>,
    I extends InputType = Parameters<Fn>[0] extends Record<string, infer II>
      ? II extends InputType
        ? II
        : UnknownInputType
      : UnknownInputType,
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
      batch: BatchTaskConfig;
    } & Omit<CreateBatchTaskWorkflowOpts<I, O>, 'fn' | 'batch'>
  ): TaskWorkflowDeclaration<I, O>;
}

/**
 * Returns the declaration factories bound to no client, so tasks can be declared where
 * a `HatchetClient` cannot exist (edge runtimes, or code that must not read
 * `HATCHET_CLIENT_TOKEN`).
 *
 * ```typescript
 * const { task, durableTask, workflow, batchTask } = declarations();
 * ```
 */
export function declarations(): Declarations {
  return {
    task: (options: any) => CreateTaskWorkflow(options),
    durableTask: (options: any) => CreateDurableTaskWorkflow(options),
    workflow: <I extends InputType, O extends StrictWorkflowOutputType>(
      options: CreateWorkflowOpts
    ) => CreateWorkflow<I, O>(options),
    batchTask: (options: any) => CreateBatchTaskWorkflow(options),
  } as Declarations;
}
