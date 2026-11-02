/**
 * The declaration factories of `@hatchet-dev/serverless`: the SDK's `task`, `durableTask`,
 * `workflow` and `batchTask` bound to no client, plus the `hatchetClient` option a non-durable
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
  CreateWorkflowDurableTaskOpts,
  CreateWorkflowOpts,
  CreateWorkflowTaskOpts,
  DurableContext,
  InputType,
  JsonObject,
  OutputType,
  StrictWorkflowOutputType,
  UnknownInputType,
} from '@hatchet-dev/typescript-sdk/edge/index.js';

export interface HatchetClientOption {
  /**
   * Whether the task talks to Hatchet at runtime through its context: child runs, streams,
   * logs to the engine, cancellation. `true` by default, and the operator then invokes the
   * task over the invocation websocket, where child results are awaited. `false` states that
   * the task is a pure function of its input: the operator delivers it over a signed POST,
   * and every client-backed member of its `ctx` throws `ServerlessLimitationError` even when
   * the handler has a `client`. A durable task always talks to Hatchet and refuses `false`.
   */
  hatchetClient?: boolean;
}

/**
 * Marks a task entry (`definition._tasks`) declared with `hatchetClient: false`. The mark is a
 * registry-wide symbol on the entry itself rather than state in this module: every published
 * entry of the package is bundled on its own, so module state would exist once per entry and
 * a task declared through `.` would be unknown to an adapter. Non-enumerable, so a copy of
 * the entry (the registration is built from copies) and the registration itself never carry
 * it; the registry matches copies to their entries by name.
 */
const PURE_TASK = Symbol.for('@hatchet-dev/serverless/pure-task');

/** Whether a task entry of a workflow definition was declared with `hatchetClient: false`. */
export function isPureTask(task: object): boolean {
  return (task as Record<symbol, unknown>)[PURE_TASK] === true;
}

function markPure(task: object): void {
  Object.defineProperty(task, PURE_TASK, { value: true, enumerable: false, configurable: true });
}

/**
 * Takes the option off a declaration's options and reports whether the task is pure. A
 * durable task refuses `hatchetClient: false`, since it always talks to Hatchet.
 */
function takeHatchetClientOption<T extends HatchetClientOption & { name: string }>(
  options: T,
  durable: boolean
): { rest: Omit<T, 'hatchetClient'>; pure: boolean } {
  const { hatchetClient, ...rest } = options;

  if (durable && hatchetClient === false) {
    throw new Error(
      `durable task "${options.name}" cannot be declared with \`hatchetClient: false\`: a durable task always talks to Hatchet, since its sleeps, waits and child runs go through the durable log over the invocation websocket. Remove the option, or declare the task with hatchet.task.`
    );
  }

  return { rest, pure: hatchetClient === false };
}

/**
 * The SDK's `WorkflowDeclaration` whose `task` takes the `hatchetClient` option.
 * `durableTask` refuses `hatchetClient: false` the way `hatchet.durableTask` does.
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
        } & HatchetClientOption)
      | TaskWorkflowDeclaration<I, TO>
  ): CreateWorkflowTaskOpts<I, TO> {
    if (options instanceof TaskWorkflowDeclaration) {
      // The standalone declaration's own entry is added, so its purity carries over.
      return super.task(options);
    }

    const { rest, pure } = takeHatchetClientOption(options, false);
    const entry = super.task(rest as never) as CreateWorkflowTaskOpts<I, TO>;

    if (pure) {
      markPure(entry);
    }

    return entry;
  }

  override durableTask<
    Name extends string,
    Fn extends (Name extends keyof O
      ? (
          input: I & MiddlewareBefore,
          ctx: DurableContext<I & MiddlewareBefore>
        ) => O[Name] extends OutputType ? O[Name] | Promise<O[Name]> : void
      : (input: I & MiddlewareBefore, ctx: DurableContext<I & MiddlewareBefore>) => void),
    FnReturn = ReturnType<Fn> extends Promise<infer P> ? P : ReturnType<Fn>,
    TO extends OutputType = Name extends keyof O
      ? O[Name] extends OutputType
        ? O[Name]
        : never
      : FnReturn extends OutputType
        ? FnReturn
        : never,
  >(
    options: Omit<CreateWorkflowTaskOpts<I, TO>, 'fn' | 'slotCost' | 'batch'> & {
      name: Name;
      fn: Fn;
    }
  ): CreateWorkflowDurableTaskOpts<I, TO> {
    const { rest } = takeHatchetClientOption(options, true);

    return super.durableTask(rest as never) as CreateWorkflowDurableTaskOpts<I, TO>;
  }
}

/**
 * The SDK's `Declarations` with the `hatchetClient` option on `task` and a `workflow` whose
 * `task` takes it too.
 */
export interface ServerlessDeclarations {
  /** Declares a task. Types come from generics or are inferred from `fn`. */
  task<I extends InputType = UnknownInputType, O extends OutputType = void>(
    options: CreateTaskWorkflowOpts<I, O> & HatchetClientOption
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
    options: { fn: Fn } & Omit<CreateTaskWorkflowOpts<I, O>, 'fn'> & HatchetClientOption
  ): TaskWorkflowDeclaration<I, O>;

  /**
   * Declares a durable task. It always talks to Hatchet over its socket, so there is no
   * `hatchetClient` option; `hatchetClient: false` throws.
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

/** The declaration factories bound to no client, with the `hatchetClient` option. */
export function declarations(): ServerlessDeclarations {
  return {
    task: (options: CreateTaskWorkflowOpts<any, any> & HatchetClientOption) => {
      const { rest, pure } = takeHatchetClientOption(options, false);
      const declaration = CreateTaskWorkflow(rest as any);

      if (pure) {
        markPure(declaration.taskDef);
      }

      return declaration;
    },
    durableTask: (options: CreateDurableTaskWorkflowOpts<any, any> & HatchetClientOption) => {
      const { rest } = takeHatchetClientOption(options, true);

      return CreateDurableTaskWorkflow(rest as any);
    },
    workflow: <I extends InputType, O extends StrictWorkflowOutputType>(
      options: CreateWorkflowOpts
    ) => new ServerlessWorkflowDeclaration<I, O>(options),
    batchTask: (options: any) => CreateBatchTaskWorkflow(options),
  } as ServerlessDeclarations;
}
