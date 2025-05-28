/* eslint-disable no-underscore-dangle */
/* eslint-disable max-classes-per-file */
import HatchetError from '@util/errors/hatchet-error';
import * as z from 'zod';
import { JsonObject } from '@bufbuild/protobuf';
import { Workflow } from './workflow';
import { Action } from './clients/dispatcher/action-listener';
import { LogLevel } from './clients/event/event-client';
import { Logger } from './util/logger';
import { parseJSON } from './util/parse';
import WorkflowRunRef from './util/workflow-run-ref';
import { WorkerLabels } from './clients/dispatcher/dispatcher-client';
import { CreateStepRateLimit, RateLimitDuration, WorkerLabelComparator } from './protoc/workflows';
import { CreateWorkflowTaskOpts, Priority } from './v1';
import {
  RunOpts,
  TaskWorkflowDeclaration,
  BaseWorkflowDeclaration as WorkflowV1,
} from './v1/declaration';
import { Conditions, Render } from './v1/conditions';
import { Action as ConditionAction } from './protoc/v1/shared/condition';
import { conditionsToPb } from './v1/conditions/transformer';
import { Duration } from './v1/client/duration';
import { JsonValue, OutputType } from './v1/types';
import { V1Worker } from './v1/client/worker/worker-internal';
import { V0Worker } from './clients/worker';
import { LegacyHatchetClient } from './clients/hatchet-client';
import { withNamespace } from './util/with-namespace';

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

export type NextStep = { [key: string]: JsonValue };

type TriggerData = Record<string, Record<string, any>>;

type ChildRunOpts = RunOpts & { key?: string; sticky?: boolean };

interface ContextData<T, K> {
  input: T;
  triggers: TriggerData;
  parents: Record<string, any>;
  triggered_by: string;
  user_data: K;
  step_run_errors: Record<string, string>;
}

export class ContextWorker {
  private worker: V0Worker | V1Worker;
  constructor(worker: V0Worker | V1Worker) {
    this.worker = worker;
  }

  /**
   * Gets the ID of the worker.
   * @returns The ID of the worker.
   */
  id() {
    return this.worker.workerId;
  }

  /**
   * Checks if the worker has a registered workflow.
   * @param workflowName - The name of the workflow to check.
   * @returns True if the workflow is registered, otherwise false.
   */
  hasWorkflow(workflowName: string) {
    return !!this.worker.workflow_registry.find((workflow) =>
      'id' in workflow ? workflow.id === workflowName : workflow.name === workflowName
    );
  }

  /**
   * Gets the current state of the worker labels.
   * @returns The labels of the worker.
   */
  labels() {
    return this.worker.labels;
  }

  /**
   * Upserts the a set of labels on the worker.
   * @param labels - The labels to upsert.
   * @returns A promise that resolves when the labels have been upserted.
   */
  upsertLabels(labels: WorkerLabels) {
    return this.worker.upsertLabels(labels);
  }
}

export class V0Context<T, K = {}> {
  data: ContextData<T, K>;
  // @deprecated use input prop instead
  input: T;
  // @deprecated use ctx.abortController instead
  controller = new AbortController();
  action: Action;
  v0: LegacyHatchetClient;

  worker: ContextWorker;

  overridesData: Record<string, any> = {};
  logger: Logger;

  spawnIndex: number = 0;

  constructor(action: Action, client: LegacyHatchetClient, worker: V0Worker | V1Worker) {
    try {
      const data = parseJSON(action.actionPayload);
      this.data = data;
      this.action = action;
      this.v0 = client;
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

  get abortController() {
    return this.controller;
  }

  get cancelled() {
    return this.controller.signal.aborted;
  }

  /**
   * Retrieves the output of a parent task.
   * @param parentTask - The a CreateTaskOpts or string of the parent task name.
   * @returns The output of the specified parent task.
   * @throws An error if the task output is not found.
   */
  async parentOutput<L extends OutputType>(parentTask: CreateWorkflowTaskOpts<any, L> | string) {
    // NOTE: parentOutput is async since we plan on potentially making this a cacheable server call
    if (typeof parentTask === 'string') {
      return this.stepOutput<L>(parentTask);
    }

    return this.stepOutput<L>(parentTask.name) as L;
  }

  /**
   * Get the output of a task.
   * @param task - The name of the task to get the output for.
   * @returns The output of the task.
   * @throws An error if the task output is not found.
   * @deprecated use ctx.parentOutput instead
   */
  stepOutput<L = NextStep>(step: string): L {
    if (!this.data.parents) {
      throw new HatchetError('Parent task outputs not found');
    }
    if (!this.data.parents[step]) {
      throw new HatchetError(`Output for parent task '${step}' not found`);
    }
    return this.data.parents[step];
  }

  /**
   * Returns errors from any task runs in the workflow.
   * @returns A record mapping task names to error messages.
   * @throws A warning if no errors are found (this method should be used in on-failure tasks).
   * @deprecated use ctx.errors() instead
   */
  stepRunErrors(): Record<string, string> {
    return this.errors();
  }

  /**
   * Returns errors from any task runs in the workflow.
   * @returns A record mapping task names to error messages.
   * @throws A warning if no errors are found (this method should be used in on-failure tasks).
   */
  errors(): Record<string, string> {
    const errors = this.data.step_run_errors || {};

    if (Object.keys(errors).length === 0) {
      this.logger.error(
        'No run errors found. `ctx.errors` is intended to be run in an on-failure task, and will only work on engine versions more recent than v0.53.10'
      );
    }

    return errors;
  }

  /**
   * Gets the dag conditional triggers for the current workflow run.
   * @returns The triggers for the current workflow.
   */
  triggers(): TriggerData {
    return this.data.triggers;
  }

  /**
   * Determines if the workflow was triggered by an event.
   * @returns True if the workflow was triggered by an event, otherwise false.
   */
  triggeredByEvent(): boolean {
    return this.data?.triggered_by === 'event';
  }

  /**
   * Gets the input data for the current workflow.
   * @returns The input data for the workflow.
   * @deprecated use task input parameter instead
   */
  workflowInput(): T {
    return this.input;
  }

  /**
   * Gets the name of the current workflow.
   * @returns The name of the workflow.
   */
  workflowName(): string {
    return this.action.jobName;
  }

  /**
   * Gets the user data associated with the workflow.
   * @returns The user data.
   */
  userData(): K {
    return this.data?.user_data;
  }

  /**
   * Gets the name of the current task.
   * @returns The name of the task.
   * @deprecated use ctx.taskName instead
   */
  stepName(): string {
    return this.taskName();
  }

  /**
   * Gets the name of the current running task.
   * @returns The name of the task.
   */
  taskName(): string {
    return this.action.stepName;
  }

  /**
   * Gets the ID of the current workflow run.
   * @returns The workflow run ID.
   */
  workflowRunId(): string {
    return this.action.workflowRunId;
  }

  /**
   * Gets the ID of the current task run.
   * @returns The task run ID.
   */
  taskRunId(): string {
    return this.action.stepRunId;
  }

  /**
   * Gets the number of times the current task has been retried.
   * @returns The retry count.
   */
  retryCount(): number {
    return this.action.retryCount;
  }

  /**
   * Logs a message from the current task.
   * @param message - The message to log.
   * @param level - The log level (optional).
   */
  log(message: string, level?: LogLevel) {
    const { stepRunId } = this.action;

    if (!stepRunId) {
      // log a warning
      this.logger.warn('cannot log from context without stepRunId');
      return;
    }

    this.v0.event.putLog(stepRunId, message, level, this.retryCount());
  }

  /**
   * Refreshes the timeout for the current task.
   * @param incrementBy - The interval by which to increment the timeout.
   * The interval should be specified in the format of '10s' for 10 seconds, '1m' for 1 minute, or '1d' for 1 day.
   */
  async refreshTimeout(incrementBy: Duration) {
    const { stepRunId } = this.action;

    if (!stepRunId) {
      // log a warning
      this.logger.warn('cannot refresh timeout from context without stepRunId');
      return;
    }

    await this.v0.dispatcher.refreshTimeout(incrementBy, stepRunId);
  }

  /**
   * Releases a worker slot for a task run such that the worker can pick up another task.
   * Note: this is an advanced feature that may lead to unexpected behavior if used incorrectly.
   * @returns A promise that resolves when the slot has been released.
   */
  async releaseSlot(): Promise<void> {
    await this.v0.dispatcher.client.releaseSlot({
      stepRunId: this.action.stepRunId,
    });
  }

  /**
   * Streams data from the current task run.
   * @param data - The data to stream (string or binary).
   * @returns A promise that resolves when the data has been streamed.
   */
  async putStream(data: string | Uint8Array) {
    const { stepRunId } = this.action;

    if (!stepRunId) {
      // log a warning
      this.logger.warn('cannot log from context without stepRunId');
      return;
    }

    await this.v0.event.putStream(stepRunId, data);
  }

  /**
   * Runs multiple children workflows in parallel without waiting for their results.
   * @param children - An array of  objects containing the workflow name, input data, and options for each workflow.
   * @returns A list of workflow run references to the enqueued runs.
   */
  async bulkRunNoWaitChildren<Q extends JsonObject = any, P extends JsonObject = any>(
    children: Array<{
      workflow: string | Workflow | WorkflowV1<Q, P>;
      input: Q;
      options?: ChildRunOpts;
    }>
  ): Promise<WorkflowRunRef<P>[]> {
    return this.spawnWorkflows(children);
  }

  /**
   * Runs multiple children workflows in parallel and waits for all results.
   * @param children - An array of objects containing the workflow name, input data, and options for each workflow.
   * @returns A list of results from the children workflows.
   */
  async bulkRunChildren<Q extends JsonObject = any, P extends JsonObject = any>(
    children: Array<{
      workflow: string | Workflow | WorkflowV1<Q, P>;
      input: Q;
      options?: ChildRunOpts;
    }>
  ): Promise<P[]> {
    const runs = await this.bulkRunNoWaitChildren(children);
    return Promise.all(runs.map((run) => run.output));
  }

  /**
   * Spawns multiple workflows.
   *
   * @param workflows - An array of objects containing the workflow name, input data, and options for each workflow.
   * @returns A list of references to the spawned workflow runs.
   * @deprecated Use bulkRunNoWaitChildren or bulkRunChildren instead.
   */
  async spawnWorkflows<Q extends JsonObject = any, P extends JsonObject = any>(
    workflows: Array<{
      workflow: string | Workflow | WorkflowV1<Q, P>;
      input: Q;
      options?: ChildRunOpts;
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

      const name = withNamespace(workflowName, this.v0.config.namespace);

      const opts = options || {};
      const { sticky } = opts;

      if (sticky && !this.worker.hasWorkflow(name)) {
        throw new HatchetError(
          `Cannot run with sticky: workflow ${name} is not registered on the worker`
        );
      }

      const resp = {
        workflowName: name,
        input,
        options: {
          ...opts,
          parentId: workflowRunId,
          parentStepRunId: stepRunId,
          childIndex: this.spawnIndex,
          desiredWorkerId: sticky ? this.worker.id() : undefined,
        },
      };
      this.spawnIndex += 1;
      return resp;
    });

    try {
      const batchSize = 100;

      let resp: WorkflowRunRef<P>[] = [];
      for (let i = 0; i < workflowRuns.length; i += batchSize) {
        const batch = workflowRuns.slice(i, i + batchSize);
        const batchResp = await this.v0.admin.runWorkflows<Q, P>(batch);
        resp = resp.concat(batchResp);
      }

      const res: WorkflowRunRef<P>[] = [];
      resp.forEach((ref, index) => {
        const wf = workflows[index].workflow;
        if (wf instanceof TaskWorkflowDeclaration) {
          // eslint-disable-next-line no-param-reassign
          ref._standaloneTaskName = wf._standalone_task_name;
        }
        res.push(ref);
      });

      return resp;
    } catch (e: any) {
      throw new HatchetError(e.message);
    }
  }

  /**
   * Runs a new workflow and waits for its result.
   *
   * @param workflow - The workflow to run (name, Workflow instance, or WorkflowV1 instance).
   * @param input - The input data for the workflow.
   * @param options - An options object containing key, sticky, priority, and additionalMetadata.
   * @returns The result of the workflow.
   */
  async runChild<Q extends JsonObject, P extends JsonObject>(
    workflow: string | Workflow | WorkflowV1<Q, P> | TaskWorkflowDeclaration<Q, P>,
    input: Q,
    options?: ChildRunOpts
  ): Promise<P> {
    const run = await this.spawnWorkflow(workflow, input, options);
    return run.output;
  }

  /**
   * Enqueues a new workflow without waiting for its result.
   *
   * @param workflow - The workflow to enqueue (name, Workflow instance, or WorkflowV1 instance).
   * @param input - The input data for the workflow.
   * @param options - An options object containing key, sticky, priority, and additionalMetadata.
   * @returns A reference to the spawned workflow run.
   */
  async runNoWaitChild<Q extends JsonObject, P extends JsonObject>(
    workflow: string | Workflow | WorkflowV1<Q, P>,
    input: Q,
    options?: ChildRunOpts
  ): Promise<WorkflowRunRef<P>> {
    return this.spawnWorkflow(workflow, input, options);
  }

  /**
   * Spawns a new workflow.
   *
   * @param workflow - The workflow to spawn (name, Workflow instance, or WorkflowV1 instance).
   * @param input - The input data for the workflow.
   * @param options - Additional options for spawning the workflow.
   * @returns A reference to the spawned workflow run.
   * @deprecated Use runChild or runNoWaitChild instead.
   */
  async spawnWorkflow<Q extends JsonObject, P extends JsonObject>(
    workflow: string | Workflow | WorkflowV1<Q, P> | TaskWorkflowDeclaration<Q, P>,
    input: Q,
    options?: ChildRunOpts
  ): Promise<WorkflowRunRef<P>> {
    const { workflowRunId, stepRunId } = this.action;

    let workflowName: string = '';

    if (typeof workflow === 'string') {
      workflowName = workflow;
    } else {
      workflowName = workflow.id;
    }

    const name = withNamespace(workflowName, this.v0.config.namespace);

    const opts = options || {};
    const { sticky } = opts;

    if (sticky && !this.worker.hasWorkflow(name)) {
      throw new HatchetError(
        `cannot run with sticky: workflow ${name} is not registered on the worker`
      );
    }

    try {
      const resp = await this.v0.admin.runWorkflow<Q, P>(name, input, {
        parentId: workflowRunId,
        parentStepRunId: stepRunId,
        childIndex: this.spawnIndex,
        desiredWorkerId: sticky ? this.worker.id() : undefined,
        ...opts,
      });

      this.spawnIndex += 1;

      if (workflow instanceof TaskWorkflowDeclaration) {
        resp._standaloneTaskName = workflow._standalone_task_name;
      }

      return resp;
    } catch (e: any) {
      throw new HatchetError(e.message);
    }
  }

  /**
   * Retrieves additional metadata associated with the current workflow run.
   * @returns A record of metadata key-value pairs.
   */
  additionalMetadata(): Record<string, string> {
    if (!this.action.additionalMetadata) {
      return {};
    }

    // parse the additional metadata
    const res: Record<string, string> = parseJSON(this.action.additionalMetadata);
    return res;
  }

  /**
   * Gets the index of this workflow if it was spawned as part of a bulk operation.
   * @returns The child index number, or undefined if not set.
   */
  childIndex(): number | undefined {
    return this.action.childWorkflowIndex;
  }

  /**
   * Gets the key associated with this workflow if it was spawned as a child workflow.
   * @returns The child key, or undefined if not set.
   */
  childKey(): string | undefined {
    return this.action.childWorkflowKey;
  }

  /**
   * Gets the ID of the parent workflow run if this workflow was spawned as a child.
   * @returns The parent workflow run ID, or undefined if not a child workflow.
   */
  parentWorkflowRunId(): string | undefined {
    return this.action.parentWorkflowRunId;
  }

  priority(): Priority | undefined {
    switch (this.action.priority) {
      case 1:
        return Priority.LOW;
      case 2:
        return Priority.MEDIUM;
      case 3:
        return Priority.HIGH;
      default:
        return undefined;
    }
  }
}

export class V0DurableContext<T, K = {}> extends V0Context<T, K> {
  waitKey: number = 0;

  /**
   * Pauses execution for the specified duration.
   * Duration is "global" meaning it will wait in real time regardless of transient failures like worker restarts.
   * @param duration - The duration to sleep for.
   * @returns A promise that resolves when the sleep duration has elapsed.
   */
  async sleepFor(duration: Duration, readableDataKey?: string) {
    return this.waitFor({ sleepFor: duration, readableDataKey });
  }

  /**
   * Pauses execution until the specified conditions are met.
   * Conditions are "global" meaning they will wait in real time regardless of transient failures like worker restarts.
   * @param conditions - The conditions to wait for.
   * @returns A promise that resolves with the event that satisfied the conditions.
   */
  async waitFor(conditions: Conditions | Conditions[]): Promise<Record<string, any>> {
    const pbConditions = conditionsToPb(Render(ConditionAction.CREATE, conditions));

    // eslint-disable-next-line no-plusplus
    const key = `waitFor-${this.waitKey++}`;
    await this.v0.durableListener.registerDurableEvent({
      taskId: this.action.stepRunId,
      signalKey: key,
      sleepConditions: pbConditions.sleepConditions,
      userEventConditions: pbConditions.userEventConditions,
    });

    const listener = this.v0.durableListener.subscribe({
      taskId: this.action.stepRunId,
      signalKey: key,
    });

    const event = await listener.get();

    // Convert event.data from Uint8Array to string if needed
    const eventData =
      event.data instanceof Uint8Array ? new TextDecoder().decode(event.data) : event.data;

    const res = JSON.parse(eventData) as Record<string, Record<string, any>>;
    return res.CREATE;
  }
}

export type StepRunFunction<T, K> = (
  ctx: V0Context<T, K>
) => Promise<NextStep | void> | NextStep | void;

/**
 * A step is a unit of work that can be run by a worker.
 * It is defined by a name, a function that returns the next step, and optional configuration.
 * @deprecated use hatchet.workflows.task factory instead
 */
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
