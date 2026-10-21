/**
 * The declaration factories of `@hatchet-dev/serverless`: the SDK's `task`, `durableTask`,
 * `workflow` and `batchTask` bound to no client, plus the `streams` option a non-durable
 * task can set. The option is stripped before the SDK sees the declaration and recorded
 * against the task entry, where the registry reads it to build the endpoint's catalog, so the
 * declarations stay the SDK's own and can also be registered by a worker.
 */
import {
  CreateBatchTaskWorkflow,
  CreateDurableTaskWorkflow,
  CreateTaskWorkflow,
  TaskWorkflowDeclaration,
  WorkflowDeclaration,
} from '@hatchet-dev/typescript-sdk/edge/index.js';
import type {
  BatchTaskConfig,
  BatchTaskFn,
  Context,
  CreateBatchTaskWorkflowOpts,
  CreateDurableTaskWorkflowOpts,
  CreateTaskWorkflowOpts,
  CreateWorkflowOpts,
  CreateWorkflowTaskOpts,
  DurableContext,
  InputType,
  JsonObject,
  OutputType,
  StrictWorkflowOutputType,
  UnknownInputType,
} from '@hatchet-dev/typescript-sdk/edge/index.js';

export interface StreamsOption {
  /**
   * Invoke this task over the invocation websocket so it can await child runs
   * (`ctx.runChild`) and other engine streams; a plain POST otherwise. The healthcheck
   * advertises it as `tasks: [{ action, streams: true }]`. Durable tasks always have a
   * socket and take no flag.
   */
  streams?: boolean;
}

/** The task entries (`definition._tasks`) declared with `streams: true`. */
const socketTasks = new WeakSet<object>();

/** Whether a task entry of a workflow definition was declared with `streams: true`. */
export function invokesOverSocket(task: object): boolean {
  return socketTasks.has(task);
}

/**
 * The SDK's `WorkflowDeclaration` whose `task` takes the `streams` option. `durableTask`
 * is inherited as is.
 */
export class ServerlessWorkflowDeclaration<
  I extends InputType = UnknownInputType,
  O extends OutputType = void,
  MiddlewareBefore extends Record<string, any> = {},
> extends WorkflowDeclaration<I, O, MiddlewareBefore> {
  override task<
    Name extends string,
    Fn extends (Name extends keyof O
      ? (
          input: I & MiddlewareBefore,
          ctx: Context<I & MiddlewareBefore>
        ) => O[Name] extends OutputType ? O[Name] | Promise<O[Name]> : void
      : (input: I & MiddlewareBefore, ctx: Context<I & MiddlewareBefore>) => void),
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
          fn?: Fn;
        } & StreamsOption)
      | TaskWorkflowDeclaration<I, TO>
  ): CreateWorkflowTaskOpts<I, TO> {
    if (options instanceof TaskWorkflowDeclaration) {
      // The standalone declaration's own entry is added, flag included.
      return super.task(options);
    }

    const { streams, ...rest } = options;
    const entry = super.task(rest as never) as CreateWorkflowTaskOpts<I, TO>;

    if (streams) {
      socketTasks.add(entry);
    }

    return entry;
  }
}

/**
 * The SDK's `Declarations` with the `streams` option on `task` and a `workflow` whose
 * `task` takes it too.
 */
export interface ServerlessDeclarations {
  /** Declares a task. Types come from generics or are inferred from `fn`. */
  task<I extends InputType = UnknownInputType, O extends OutputType = void>(
    options: CreateTaskWorkflowOpts<I, O> & StreamsOption
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
    options: { fn: Fn } & Omit<CreateTaskWorkflowOpts<I, O>, 'fn'> & StreamsOption
  ): TaskWorkflowDeclaration<I, O>;

  /** Declares a durable task. It always runs on a socket, so there is no `streams` option. */
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

  /** Declares a workflow: a DAG of tasks added with `workflow.task(...)`. */
  workflow<I extends InputType = UnknownInputType, O extends StrictWorkflowOutputType = {}>(
    options: CreateWorkflowOpts
  ): ServerlessWorkflowDeclaration<I, O>;

  /**
   * Declares a batch task. The handler registers it as a plain task, since batch tasks need
   * a worker; see LIMITATIONS.md.
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

/** The declaration factories bound to no client, with the `streams` option. */
export function declarations(): ServerlessDeclarations {
  return {
    task: (options: CreateTaskWorkflowOpts<any, any> & StreamsOption) => {
      const { streams, ...rest } = options;
      const declaration = CreateTaskWorkflow(rest as any);

      if (streams) {
        socketTasks.add(declaration.taskDef);
      }

      return declaration;
    },
    durableTask: (options: any) => CreateDurableTaskWorkflow(options),
    workflow: <I extends InputType, O extends StrictWorkflowOutputType>(
      options: CreateWorkflowOpts
    ) => new ServerlessWorkflowDeclaration<I, O>(options),
    batchTask: (options: any) => CreateBatchTaskWorkflow(options),
  } as ServerlessDeclarations;
}
