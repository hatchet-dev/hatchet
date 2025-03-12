/* eslint-disable max-classes-per-file */
import HatchetError from '@util/errors/hatchet-error';
import * as z from 'zod';
import { Workflow } from './workflow';
import { Action } from './clients/dispatcher/action-listener';
import { LogLevel } from './clients/event/event-client';
import { Logger } from './util/logger';
import { parseJSON } from './util/parse';
import { InternalHatchetClient } from './clients/hatchet-client';
import WorkflowRunRef from './util/workflow-run-ref';
import { Worker } from './clients/worker';
import { WorkerLabels } from './clients/dispatcher/dispatcher-client';
import { CreateStepRateLimit, RateLimitDuration, WorkerLabelComparator } from './protoc/workflows';
import { CreateTaskOpts } from './v1/task';
import { Workflow as WorkflowV1 } from './v1/workflow';

export const CreateRateLimitSchema = z.object({
  key: z.string().optional(),
  staticKey: z.string().optional(),
  dynamicKey: z.string().optional(),

  units: z.union([z.number().min(0), z.string()]),
  limit: z.union([z.number().min(1), z.string()]).optional(),
  duration: z.nativeEnum(RateLimitDuration).optional(),
});

export const DesiredWorkerLabelSchema = z
  .union([
    z.string(),
    z.number().int(),
    z.object({
      value: z.union([z.string(), z.number()]),
      required: z.boolean().optional(),
      weight: z.number().int().optional(),

      // (optional) comparator for the label
      // if not provided, the default is EQUAL
      // desired COMPARATOR actual (i.e. desired > actual for GREATER_THAN)
      comparator: z.nativeEnum(WorkerLabelComparator).optional(),
    }),
  ])
  .optional();

export const CreateStepSchema = z.object({
  name: z.string(),
  parents: z.array(z.string()).optional(),
  timeout: z.string().optional(),
  retries: z.number().optional(),
  rate_limits: z.array(CreateRateLimitSchema).optional(),
  worker_labels: z.record(z.lazy(() => DesiredWorkerLabelSchema)).optional(),
  backoff: z
    .object({
      factor: z.number().optional(),
      maxSeconds: z.number().optional(),
    })
    .optional(),
});

export type JsonObject = { [Key in string]: JsonValue } & {
  [Key in string]?: JsonValue | undefined;
};

export type JsonArray = JsonValue[] | readonly JsonValue[];

export type JsonPrimitive = string | number | boolean | null;

export type JsonValue = JsonPrimitive | JsonObject | JsonArray;

export type NextStep = { [key: string]: JsonValue };

interface ContextData<T, K> {
  input: T;
  parents: Record<string, any>;
  triggered_by: string;
  user_data: K;
  step_run_errors: Record<string, string>;
}

export class ContextWorker {
  private worker: Worker;
  constructor(worker: Worker) {
    this.worker = worker;
  }

  id() {
    return this.worker.workerId;
  }

  hasWorkflow(workflowName: string) {
    return !!this.worker.workflow_registry.find((workflow) => workflow.id === workflowName);
  }

  labels() {
    return this.worker.labels;
  }

  upsertLabels(labels: WorkerLabels) {
    return this.worker.upsertLabels(labels);
  }
}

export class Context<T, K = {}> {
  data: ContextData<T, K>;
  input: T;
  controller = new AbortController();
  action: Action;
  client: InternalHatchetClient;

  worker: ContextWorker;

  overridesData: Record<string, any> = {};
  logger: Logger;

  spawnIndex: number = 0;

  constructor(action: Action, client: InternalHatchetClient, worker: Worker) {
    try {
      const data = parseJSON(action.actionPayload);
      this.data = data;
      this.action = action;
      this.client = client;
      this.worker = new ContextWorker(worker);
      this.logger = client.config.logger(`Context Logger`, client.config.log_level);

      // if this is a getGroupKeyRunId, the data is the workflow input
      if (action.getGroupKeyRunId !== '') {
        this.input = data;
      } else {
        this.input = data.input;
      }

      this.overridesData = data.overrides || {};
    } catch (e: any) {
      throw new HatchetError(`Could not parse payload: ${e.message}`);
    }
  }

  // NOTE: parentData is async since we plan on potentially making this a cacheable server call
  async parentData<L = NextStep>(task: CreateTaskOpts<any, L> | string) {
    if (typeof task === 'string') {
      return this.stepOutput<L>(task);
    }

    return this.stepOutput<L>(task.name) as L;
  }

  // TODO deprecated
  stepOutput<L = NextStep>(step: string): L {
    if (!this.data.parents) {
      throw new HatchetError('Step output not found');
    }
    if (!this.data.parents[step]) {
      throw new HatchetError(`Step output for '${step}' not found`);
    }
    return this.data.parents[step];
  }

  stepRunErrors(): Record<string, string> {
    const errors = this.data.step_run_errors || {};

    if (Object.keys(errors).length === 0) {
      this.logger.error(
        'No step run errors found. `ctx.stepRunErrors` is intended to be run in an on-failure step, and will only work on engine versions more recent than v0.53.10'
      );
    }

    return errors;
  }

  triggeredByEvent(): boolean {
    return this.data?.triggered_by === 'event';
  }

  workflowInput(): T {
    return this.input;
  }

  workflowName(): string {
    return this.action.jobName;
  }

  userData(): K {
    return this.data?.user_data;
  }

  stepName(): string {
    return this.action.stepName;
  }

  workflowRunId(): string {
    return this.action.workflowRunId;
  }

  retryCount(): number {
    return this.action.retryCount;
  }

  playground(name: string, defaultValue: string = ''): string {
    if (name in this.overridesData) {
      return this.overridesData[name];
    }

    this.client.dispatcher.putOverridesData({
      stepRunId: this.action.stepRunId,
      path: name,
      value: JSON.stringify(defaultValue),
    });

    return defaultValue;
  }

  log(message: string, level?: LogLevel) {
    const { stepRunId } = this.action;

    if (!stepRunId) {
      // log a warning
      this.logger.warn('cannot log from context without stepRunId');
      return;
    }

    this.client.event.putLog(stepRunId, message, level);
  }

  /**
   * Refreshes the timeout for the current step.
   * @param incrementBy - The interval by which to increment the timeout.
   *                     The interval should be specified in the format of '10s' for 10 seconds,
   *                     '1m' for 1 minute, or '1d' for 1 day.
   */
  async refreshTimeout(incrementBy: string) {
    const { stepRunId } = this.action;

    if (!stepRunId) {
      // log a warning
      this.logger.warn('cannot refresh timeout from context without stepRunId');
      return;
    }

    await this.client.dispatcher.refreshTimeout(incrementBy, stepRunId);
  }

  async releaseSlot(): Promise<void> {
    await this.client.dispatcher.client.releaseSlot({
      stepRunId: this.action.stepRunId,
    });
  }

  async putStream(data: string | Uint8Array) {
    const { stepRunId } = this.action;

    if (!stepRunId) {
      // log a warning
      this.logger.warn('cannot log from context without stepRunId');
      return;
    }

    await this.client.event.putStream(stepRunId, data);
  }

  /**
   * Enqueues multiple children workflows in parallel.
   * @param children an array of objects containing the workflow name, input data, and options for each workflow
   * @returns a list of workflow run references to the enqueued runs
   */
  bulkEnqueueChildren<Q extends Record<string, any> = any, P extends Record<string, any> = any>(
    children: Array<{
      workflow: string | Workflow | WorkflowV1<Q, P>;
      input: Q;
      options?: {
        key?: string;
        sticky?: boolean;
        additionalMetadata?: Record<string, string>;
      };
    }>
  ): Promise<WorkflowRunRef<P>[]> {
    return this.spawnWorkflows(children);
  }

  /**
   * Runs multiple children workflows in parallel.
   * @param children an array of objects containing the workflow name, input data, and options for each workflow
   * @returns a list of results from the children workflows
   */
  async bulkRunChildren<Q extends Record<string, any> = any, P extends Record<string, any> = any>(
    children: Array<{
      workflow: string | Workflow | WorkflowV1<Q, P>;
      input: Q;
      options?: {
        key?: string;
        sticky?: boolean;
        additionalMetadata?: Record<string, string>;
      };
    }>
  ): Promise<P[]> {
    const runs = await this.bulkEnqueueChildren(children);
    const res = runs.map((run) => run.result());
    return Promise.all(res);
  }

  /**
   * Spawns multiple workflows.
   *
   * @param workflows an array of objects containing the workflow name, input data, and options for each workflow
   * @returns a list of references to the spawned workflow runs
   * @deprecated use bulkEnqueueChildren or bulkRunChildren instead
   */
  spawnWorkflows<Q extends Record<string, any> = any, P extends Record<string, any> = any>(
    workflows: Array<{
      workflow: string | Workflow | WorkflowV1<Q, P>;
      input: Q;
      options?: {
        key?: string;
        sticky?: boolean;
        additionalMetadata?: Record<string, string>;
      };
    }>
  ): Promise<WorkflowRunRef<P>[]> {
    const { workflowRunId, stepRunId } = this.action;

    const workflowRuns = workflows.map(({ workflow, input, options }) => {
      let workflowName: string;

      if (typeof workflow === 'string') {
        workflowName = workflow;
      } else {
        workflowName = workflow.id;
      }

      const name = this.client.config.namespace + workflowName;

      let key: string | undefined;
      let sticky: boolean | undefined = false;
      let metadata: Record<string, string> | undefined;

      if (options) {
        key = options.key;
        sticky = options.sticky;
        metadata = options.additionalMetadata;
      }

      if (sticky && !this.worker.hasWorkflow(name)) {
        throw new HatchetError(
          `Cannot run with sticky: workflow ${name} is not registered on the worker`
        );
      }

      const resp = {
        workflowName: name,
        input,
        options: {
          parentId: workflowRunId,
          parentStepRunId: stepRunId,
          childKey: key,
          childIndex: this.spawnIndex,
          desiredWorkerId: sticky ? this.worker.id() : undefined,
          additionalMetadata: metadata,
        },
      };
      this.spawnIndex += 1;
      return resp;
    });

    try {
      const resp = this.client.admin.runWorkflows<Q, P>(workflowRuns);

      return resp;
    } catch (e: any) {
      throw new HatchetError(e.message);
    }
  }

  /**
   * Runs a new workflow.
   *
   * @param workflow the workflow to run
   * @param input the input data for the workflow
   * @param options additional options for spawning the workflow. If a string is provided, it is used as the key.
   * @param <Q> the type of the input data
   * @param <P> the type of the output data
   * @return the result of the workflow
   */
  async runChild<Q extends Record<string, any>, P extends Record<string, any>>(
    workflow: string | Workflow | WorkflowV1<Q, P>,
    input: Q,
    options?:
      | string
      | { key?: string; sticky?: boolean; additionalMetadata?: Record<string, string> }
  ): Promise<P> {
    const run = await this.spawnWorkflow(workflow, input, options);
    return run.result();
  }

  /**
   * Enqueues a new workflow.
   *
   * @param workflowName the name of the workflow to spawn
   * @param input the input data for the workflow
   * @param options additional options for spawning the workflow. If a string is provided, it is used as the key.
   *                If an object is provided, it can include:
   *                - key: a unique identifier for the workflow (deprecated, use options.key instead)
   *                - sticky: a boolean indicating whether to use sticky execution
   * @param <Q> the type of the input data
   * @param <P> the type of the output data
   * @return a reference to the spawned workflow run
   */
  enqueueChild<Q extends Record<string, any>, P extends Record<string, any>>(
    workflow: string | Workflow | WorkflowV1<Q, P>,
    input: Q,
    options?:
      | string
      | { key?: string; sticky?: boolean; additionalMetadata?: Record<string, string> }
  ): WorkflowRunRef<P> {
    return this.spawnWorkflow(workflow, input, options);
  }

  /**
   * Spawns a new workflow.
   *
   * @param workflowName the name of the workflow to spawn
   * @param input the input data for the workflow
   * @param options additional options for spawning the workflow. If a string is provided, it is used as the key.
   *                If an object is provided, it can include:
   *                - key: a unique identifier for the workflow (deprecated, use options.key instead)
   *                - sticky: a boolean indicating whether to use sticky execution
   * @param <Q> the type of the input data
   * @param <P> the type of the output data
   * @return a reference to the spawned workflow run
   * @deprecated use runChild or enqueueChild instead
   */
  spawnWorkflow<Q extends Record<string, any>, P extends Record<string, any>>(
    workflow: string | Workflow | WorkflowV1<Q, P>,
    input: Q,
    options?:
      | string
      | { key?: string; sticky?: boolean; additionalMetadata?: Record<string, string> }
  ): WorkflowRunRef<P> {
    const { workflowRunId, stepRunId } = this.action;

    let workflowName: string = '';

    if (typeof workflow === 'string') {
      workflowName = workflow;
    } else {
      workflowName = workflow.id;
    }

    const name = this.client.config.namespace + workflowName;

    let key: string | undefined = '';
    let sticky: boolean | undefined = false;
    let metadata: Record<string, string> | undefined;

    if (typeof options === 'string') {
      this.logger.warn(
        'Using key param is deprecated and will be removed in a future release. Use options.key instead.'
      );
      key = options;
    } else {
      key = options?.key;
      sticky = options?.sticky;
      metadata = options?.additionalMetadata;
    }

    if (sticky && !this.worker.hasWorkflow(name)) {
      throw new HatchetError(
        `cannot run with sticky: workflow ${name} is not registered on the worker`
      );
    }

    try {
      const resp = this.client.admin.runWorkflow<Q, P>(name, input, {
        parentId: workflowRunId,
        parentStepRunId: stepRunId,
        childKey: key,
        childIndex: this.spawnIndex,
        desiredWorkerId: sticky ? this.worker.id() : undefined,
        additionalMetadata: metadata,
      });

      this.spawnIndex += 1;

      return resp;
    } catch (e: any) {
      throw new HatchetError(e.message);
    }
  }

  additionalMetadata(): Record<string, string> {
    if (!this.action.additionalMetadata) {
      return {};
    }

    // parse the additional metadata
    const res: Record<string, string> = parseJSON(this.action.additionalMetadata);
    return res;
  }

  childIndex(): number | undefined {
    return this.action.childWorkflowIndex;
  }

  childKey(): string | undefined {
    return this.action.childWorkflowKey;
  }

  parentWorkflowRunId(): string | undefined {
    return this.action.parentWorkflowRunId;
  }
}

export type StepRunFunction<T, K> = (
  ctx: Context<T, K>
) => Promise<NextStep | void> | NextStep | void;

export interface CreateStep<T, K> extends z.infer<typeof CreateStepSchema> {
  run: StepRunFunction<T, K>;
}

export function mapRateLimit(limits: CreateStep<any, any>['rate_limits']): CreateStepRateLimit[] {
  if (!limits) return [];

  return limits.map((l) => {
    let key = l.staticKey;
    const keyExpression = l.dynamicKey;

    if (l.key !== undefined) {
      // eslint-disable-next-line no-console
      console.warn(
        'key is deprecated and will be removed in a future release, please use staticKey instead'
      );
      key = l.key;
    }

    if (keyExpression !== undefined) {
      if (key !== undefined) {
        throw new Error('Cannot have both static key and dynamic key set');
      }
      key = keyExpression;
      if (!validateCelExpression(keyExpression)) {
        throw new Error(`Invalid CEL expression: ${keyExpression}`);
      }
    }

    if (key === undefined) {
      throw new Error(`Invalid key`);
    }

    let units: number | undefined;
    let unitsExpression: string | undefined;
    if (typeof l.units === 'number') {
      units = l.units;
    } else {
      if (!validateCelExpression(l.units)) {
        throw new Error(`Invalid CEL expression: ${l.units}`);
      }
      unitsExpression = l.units;
    }

    let limitExpression: string | undefined;
    if (l.limit !== undefined) {
      if (typeof l.limit === 'number') {
        limitExpression = `${l.limit}`;
      } else {
        if (!validateCelExpression(l.limit)) {
          throw new Error(`Invalid CEL expression: ${l.limit}`);
        }
        limitExpression = l.limit;
      }
    }

    if (keyExpression !== undefined && limitExpression === undefined) {
      throw new Error('CEL based keys requires limit to be set');
    }

    return {
      key,
      keyExpr: keyExpression,
      units,
      unitsExpr: unitsExpression,
      limitValuesExpr: limitExpression,
      duration: l.duration,
    };
  });
}

// Helper function to validate CEL expressions
function validateCelExpression(expr: string): boolean {
  // This is a placeholder. In a real implementation, you'd need to use a CEL parser or validator.
  // For now, we'll just return true to mimic the behavior.
  return true;
}
