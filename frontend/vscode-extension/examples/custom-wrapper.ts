/**
 * Example: custom workflow builder wrapper.
 *
 * Annotate any factory function that returns a workflow-builder-like object
 * with `@hatchet-workflow`.  The VS Code extension will then detect variables
 * created by this function and render their DAGs, just like standard
 * `hatchet.workflow(...)` calls.
 *
 * Optional tags:
 *   @hatchet-task-method  — name of the method used to register tasks
 *                           (defaults to "task")
 *   @hatchet-task-parents — name of the option property that lists parent
 *                           tasks (defaults to "parents")
 */
import Hatchet, {
  type Context,
  type WorkflowDeclaration,
} from '@hatchet-dev/typescript-sdk';
import type { CreateWorkflowTaskOpts } from '@hatchet-dev/typescript-sdk';

// ── Supporting types ──────────────────────────────────────────────────────────

type JsonObject = Record<string, unknown>;

interface Services {
  db: unknown; // replace with your actual DB/service client types
}

interface TaskOptions<TInput extends JsonObject, TOutput extends JsonObject> {
  /** Business logic to execute for this task. */
  fn: (args: { input: TInput; services: Services }) => Promise<TOutput>;
  /** Tasks that must complete before this one runs. */
  parents?: TaskHandle<JsonObject>[];
  retries?: number;
  timeoutSeconds?: number;
}

/**
 * A lightweight handle returned by `WorkflowBuilder.task()`.
 * Pass these as `parents` to express DAG dependencies.
 */
export interface TaskHandle<TOutput extends JsonObject> {
  /** Display name shown in the DAG. */
  readonly name: string;
  /** @internal Raw Hatchet task opts — used to wire up parents internally. */
  readonly _taskDef: CreateWorkflowTaskOpts<any, any>;
}

interface WorkflowBuilderOptions<TInput extends JsonObject> {
  /** Workflow display name (shown in the Hatchet UI and the VS Code DAG view). */
  name: string;
  version?: string;
  /** Called once per task invocation to produce scoped service clients. */
  createServices: (input: TInput) => Promise<Services>;
}

/**
 * A fluent builder for Hatchet workflows with injected service context.
 *
 * Usage:
 * ```ts
 * const wf = createWorkflowBuilder({ name: 'my-workflow', createServices });
 * const step1 = wf.task('step1', { fn: async ({ input, services }) => ... });
 * const step2 = wf.task('step2', { parents: [step1], fn: async (...) => ... });
 * export default wf.build();
 * ```
 */
export interface WorkflowBuilder<TInput extends JsonObject> {
  /**
   * Register a task on this workflow.
   *
   * @param name    Unique task name shown in the DAG.
   * @param options Task configuration including the `fn` and optional `parents`.
   */
  task<TOutput extends JsonObject>(
    name: string,
    options: TaskOptions<TInput, TOutput>,
  ): TaskHandle<TOutput>;

  /** Finalise the workflow and return the underlying `WorkflowDeclaration`. */
  build(): WorkflowDeclaration;
}

// ── Factory function ──────────────────────────────────────────────────────────

const hatchet = Hatchet.init();

/**
 * Create a workflow builder that automatically injects service context into
 * every task function.
 *
 * Annotated with `@hatchet-workflow` so the VS Code extension can detect
 * variables created by this function and render their task graphs.
 *
 * @hatchet-workflow
 * @hatchet-task-method task
 * @hatchet-task-parents parents
 */
export function createWorkflowBuilder<TInput extends JsonObject>(
  options: WorkflowBuilderOptions<TInput>,
): WorkflowBuilder<TInput> {
  const wf = hatchet.workflow<TInput>({ name: options.name });

  return {
    task<TOutput extends JsonObject>(
      name: string,
      taskOpts: TaskOptions<TInput, TOutput>,
    ): TaskHandle<TOutput> {
      const wrappedFn = async (input: TInput, _ctx: Context<TInput>): Promise<TOutput> => {
        const services = await options.createServices(input);
        return taskOpts.fn({ input, services });
      };

      const taskDef = wf.task({
        name,
        retries: taskOpts.retries,
        ...(taskOpts.timeoutSeconds
          ? { executionTimeout: `${taskOpts.timeoutSeconds}s` }
          : {}),
        parents: taskOpts.parents?.map((p) => p._taskDef),
        fn: wrappedFn,
      });

      return { name, _taskDef: taskDef };
    },

    build(): WorkflowDeclaration {
      return wf;
    },
  };
}
