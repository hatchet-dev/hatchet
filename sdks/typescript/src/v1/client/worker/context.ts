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
import { ContextWorker, NextStep } from '@hatchet/step';
import { V1Worker } from './worker-internal';
import { Duration } from '../duration';

type TriggerData = Record<string, Record<string, any>>;

type ChildRunOpts = RunOpts & { key?: string; sticky?: boolean };

type LogExtra = {
  extra?: any;
  error?: Error;
};

interface ContextData<T, K> {
  input: T;
  triggers: TriggerData;
  parents: Record<string, any>;
  triggered_by: string;
  user_data: K;
  step_run_errors: Record<string, string>;
}

export class Context<T, K = {}> {
  data: ContextData<T, K>;
  // @deprecated use input prop instead
  input: T;

  // @deprecated use ctx.abortController instead
  controller = new AbortController();
  action: Action;
  v1: HatchetClient;

  worker: ContextWorker;

  overridesData: Record<string, any> = {};
  _logger: Logger;

  spawnIndex: number = 0;

  constructor(action: Action, v1: HatchetClient, worker: V1Worker) {
    try {
      const data = parseJSON(action.actionPayload);
      this.data = data;
      this.action = action;
      this.v1 = v1;
      this.worker = new ContextWorker(worker);
      this._logger = v1.config.logger(`Context Logger`, v1.config.log_level);

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
      this._logger.error(
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
  log(message: string, level?: LogLevel, extra?: LogExtra) {
    const { stepRunId } = this.action;

    if (!stepRunId) {
      // log a warning
      this._logger.warn('cannot log from context without stepRunId');
      return;
    }

    const logger = this.v1.config.logger('ctx', this.v1.config.log_level);
    const contextExtra = {
      workflowRunId: this.action.workflowRunId,
      taskRunId: this.action.stepRunId,
      retryCount: this.action.retryCount,
      workflowName: this.action.jobName,
      ...extra?.extra,
    };

    if (!level || level === 'INFO') {
      logger.info(message, contextExtra);
    } else if (level === 'DEBUG') {
      logger.debug(message, contextExtra);
    } else if (level === 'WARN') {
      logger.warn(message, extra?.error, contextExtra);
    } else if (level === 'ERROR') {
      logger.error(message, extra?.error, contextExtra);
    }

    // FIXME: this is a hack to get around the fact that the log level is not typed
    this.v1.event.putLog(stepRunId, message, level as any);
  }

  get logger() {
    return {
      info: (message: string, extra?: any) => {
        this.log(message, 'INFO', { extra });
      },
      debug: (message: string, extra?: any) => {
        this.log(message, 'DEBUG', { extra });
      },
      warn: (message: string, extra?: LogExtra) => {
        this.log(message, 'WARN', extra);
      },
      error: (message: string, extra?: LogExtra) => {
        this.log(message, 'ERROR', extra);
      },
      trace: (message: string, extra?: LogExtra) => {
        const logger = this.v1.config.logger('ctx', this.v1.config.log_level);
        logger.trace(message, extra);
      },
    };
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
      this._logger.warn('cannot refresh timeout from context without stepRunId');
      return;
    }

    await this.v1._v0.dispatcher.refreshTimeout(incrementBy, stepRunId);
  }

  /**
   * Releases a worker slot for a task run such that the worker can pick up another task.
   * Note: this is an advanced feature that may lead to unexpected behavior if used incorrectly.
   * @returns A promise that resolves when the slot has been released.
   */
  async releaseSlot(): Promise<void> {
    await this.v1._v0.dispatcher.client.releaseSlot({
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
      this._logger.warn('cannot log from context without stepRunId');
      return;
    }

    await this.v1._v0.event.putStream(stepRunId, data);
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
      _standaloneTaskName:
        workflow instanceof TaskWorkflowDeclaration ? workflow._standalone_task_name : undefined,
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
    return this.v1.admin.runWorkflow<Q, P>(workflowName, input, opts);
  }

  private spawnBulk<Q extends JsonObject, P extends JsonObject>(
    children: Array<{
      workflow: string | Workflow | WorkflowV1<Q, P>;
      input: Q;
      options?: ChildRunOpts;
    }>
  ) {
    const workflows: Parameters<typeof this.v1.admin.runWorkflows<Q, P>>[0] = children.map(
      (child) => {
        const { workflowName, opts } = this.spawnOptions(child.workflow, child.options);
        return { workflowName, input: child.input, options: opts };
      }
    );

    return this.v1.admin.runWorkflows<Q, P>(workflows);
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
    const run = await this.spawn(workflow, input, options);
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
    const ref = await this.spawn(workflow, input, options);
    return ref;
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
  // FIXME: drop these at some point soon

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
   * Gets the input data for the current workflow.
   * @returns The input data for the workflow.
   * @deprecated use task input parameter instead
   */
  workflowInput(): T {
    return this.input;
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

      const name = this.v1.config.namespace + workflowName;

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
      const batchSize = 10;

      let resp: WorkflowRunRef<P>[] = [];
      for (let i = 0; i < workflowRuns.length; i += batchSize) {
        const batch = workflowRuns.slice(i, i + batchSize);
        const batchResp = await this.v1._v0.admin.runWorkflows<Q, P>(batch);
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

    const name = this.v1.config.namespace + workflowName;

    const opts = options || {};
    const { sticky } = opts;

    if (sticky && !this.worker.hasWorkflow(name)) {
      throw new HatchetError(
        `cannot run with sticky: workflow ${name} is not registered on the worker`
      );
    }

    try {
      const resp = await this.v1._v0.admin.runWorkflow<Q, P>(name, input, {
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
    await this.v1._v0.durableListener.registerDurableEvent({
      taskId: this.action.stepRunId,
      signalKey: key,
      sleepConditions: pbConditions.sleepConditions,
      userEventConditions: pbConditions.userEventConditions,
    });

    const listener = this.v1._v0.durableListener.subscribe({
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
