/* eslint-disable no-underscore-dangle */
/* eslint-disable max-classes-per-file */
import {
  Priority,
  RunOpts,
  TaskWorkflowDeclaration,
  BaseWorkflowDeclaration as WorkflowV1,
} from '@hatchet/v1/declaration';
import HatchetError from '@util/errors/hatchet-error';
import { JsonObject } from '@bufbuild/protobuf';
import { Action } from '@hatchet/clients/dispatcher/action-listener';
import { Logger, LogLevel } from '@hatchet/util/logger';
import { parseJSON } from '@hatchet/util/parse';
import WorkflowRunRef from '@hatchet/util/workflow-run-ref';
import { Conditions, Render } from '@hatchet/v1/conditions';
import { conditionsToPb } from '@hatchet/v1/conditions/transformer';
import { CreateWorkflowTaskOpts } from '@hatchet/v1/task';
import { OutputType } from '@hatchet/v1/types';
import { Workflow } from '@hatchet/workflow';
import { Action as ConditionAction } from '@hatchet/protoc/v1/shared/condition';
import { HatchetClient } from '@hatchet/v1';
import { ContextWorker, V0Context } from '@hatchet/step';
import { V1Worker } from './worker-internal';
import { Duration } from '../duration';

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

export class Context<T, K = {}> extends V0Context<T, K> {
  data: ContextData<T, K>;
  input: T;

  controller = new AbortController();
  action: Action;
  v1: HatchetClient;

  worker: ContextWorker;

  overridesData: Record<string, any> = {};
  logger: Logger;

  spawnIndex: number = 0;

  constructor(action: Action, v1: HatchetClient, worker: V1Worker) {
    super(action, v1._v0, worker);

    try {
      const data = parseJSON(action.actionPayload);
      this.data = data;
      this.action = action;
      this.v1 = v1;
      this.worker = new ContextWorker(worker);
      this.logger = v1.config.logger(`Context Logger`, v1.config.log_level);

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
  async parentOutput<L extends OutputType>(
    parentTask: CreateWorkflowTaskOpts<any, L> | string
  ): Promise<L> {
    // NOTE: parentOutput is async since we plan on potentially making this a cacheable server call
    if (typeof parentTask !== 'string') {
      return this.parentOutput<L>(parentTask.name);
    }

    if (!this.data.parents) {
      throw new HatchetError('Parent task outputs not found');
    }
    if (!this.data.parents[parentTask]) {
      throw new HatchetError(`Output for parent task '${parentTask}' not found`);
    }
    return this.data.parents[parentTask] as L;
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

    // FIXME: this is a hack to get around the fact that the log level is not typed
    this.v0.event.putLog(stepRunId, message, level as any);
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

  private spawnOptions(workflow: string | Workflow | WorkflowV1<any, any>, options?: ChildRunOpts) {
    let workflowName: string;

    if (typeof workflow === 'string') {
      workflowName = workflow;
    } else {
      workflowName = workflow.id;
    }

    const opts = options || {};
    const { sticky } = opts;

    if (sticky && !this.worker.hasWorkflow(workflowName)) {
      throw new HatchetError(
        `Cannot run with sticky: workflow ${workflowName} is not registered on the worker`
      );
    }

    const { workflowRunId, stepRunId } = this.action;

    const finalOpts = {
      ...options,
      parentId: workflowRunId,
      parentStepRunId: stepRunId,
      childIndex: this.spawnIndex,
      desiredWorkerId: sticky ? this.worker.id() : undefined,
    };

    this.spawnIndex += 1;

    return { workflowName, opts: finalOpts };
  }

  private spawn<Q extends JsonObject, P extends JsonObject>(
    workflow: string | Workflow | WorkflowV1<Q, P>,
    input: Q,
    options?: ChildRunOpts
  ) {
    const { workflowName, opts } = this.spawnOptions(workflow, options);
    return this.v0.admin.runWorkflow<Q, P>(workflowName, input, opts);
  }

  private spawnBulk<Q extends JsonObject, P extends JsonObject>(
    children: Array<{
      workflow: string | Workflow | WorkflowV1<Q, P>;
      input: Q;
      options?: ChildRunOpts;
    }>
  ) {
    const workflows: Parameters<typeof this.v0.admin.runWorkflows<Q, P>>[0] = children.map(
      (child) => {
        const { workflowName, opts } = this.spawnOptions(child.workflow, child.options);
        return { workflowName, input: child.input, options: opts };
      }
    );

    return this.v0.admin.runWorkflows<Q, P>(workflows);
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
    return this.spawnBulk<Q, P>(children);
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

export class DurableContext<T, K = {}> extends Context<T, K> {
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
