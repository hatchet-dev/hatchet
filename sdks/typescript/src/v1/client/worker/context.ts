/**
 * The Hatchet Context class provides helper methods and useful data to tasks at runtime. It is passed as the second argument to all tasks and durable tasks.
 *
 * There are two types of context classes you'll encounter:
 *
 * - Context - The standard context for regular tasks with methods for logging, task output retrieval, cancellation, and more.
 * - DurableContext - An extended context for durable tasks that includes additional methods for durable execution.
 * @module Context
 */

import {
  Priority,
  RunOpts,
  TaskWorkflowDeclaration,
  BaseWorkflowDeclaration as WorkflowV1,
} from '@hatchet/v1/declaration';
import HatchetError from '@util/errors/hatchet-error';
import { Action } from '@hatchet/clients/dispatcher/action-listener';
import { Logger, LogLevel } from '@hatchet/util/logger';
import { parseJSON } from '@hatchet/util/parse';
import WorkflowRunRef from '@hatchet/util/workflow-run-ref';
import { Conditions, Render } from '@hatchet/v1/conditions';
import { conditionsToPb } from '@hatchet/v1/conditions/transformer';
import { CreateWorkflowDurableTaskOpts, CreateWorkflowTaskOpts } from '@hatchet/v1/task';
import { JsonObject, OutputType } from '@hatchet/v1/types';
import { Action as ConditionAction } from '@hatchet/protoc/v1/shared/condition';
import { HatchetClient } from '@hatchet/v1';
import { applyNamespace } from '@hatchet/util/apply-namespace';
import { createAbortError, rethrowIfAborted } from '@hatchet/util/abort-error';
import { WorkerLabels } from '@hatchet/clients/dispatcher/dispatcher-client';
import { parentRunContextManager } from '@hatchet/v1/parent-run-context-vars';
import { NextStep } from '@hatchet-dev/typescript-sdk/legacy/step';
import { DurableListenerClient } from '@hatchet/clients/listeners/durable-listener/durable-listener-client';
import { createHash } from 'crypto';
import { z } from 'zod/v4';
import { InternalWorker } from './worker-internal';
import { Duration, durationToMs, durationToString } from '../duration';
import { DurableEvictionManager } from './eviction/eviction-manager';
import { ActionKey } from './eviction/eviction-cache';
import { supportsEviction } from './engine-version';
import { waitForPreEviction } from './deprecated/pre-eviction';
// TODO remove this once we have a proper next step type

type TriggerData = Record<string, Record<string, any>>;

type ChildRunOpts = RunOpts & { key?: string; sticky?: boolean };

export interface SleepResult {
  /** The sleep duration in milliseconds. */
  durationMs: number;
}

export interface SleepForOptions {
  /** Optional key used for condition result indexing. */
  readableDataKey?: string;
  /** Optional human-readable wait label shown in dashboard run details. */
  label?: string;
}

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

/**
 * ContextWorker is a wrapper around the V1Worker class that provides a more user-friendly interface for the worker from the context of a run.
 */
export class ContextWorker {
  private worker: InternalWorker;
  constructor(worker: InternalWorker) {
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

  /** @deprecated — kept for backward compat; prefer {@link nextChildIndex}. */
  spawnIndex: number = 0;
  streamIndex = 0;

  protected nextChildIndex(n = 1): number {
    const ctx = parentRunContextManager.getContext();
    const idx = ctx?.childIndex ?? this.spawnIndex;
    parentRunContextManager.incrementChildIndex(n);
    this.spawnIndex = idx + n;
    return idx;
  }

  constructor(action: Action, v1: HatchetClient, worker: InternalWorker) {
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

  protected throwIfCancelled(): void {
    if (this.abortController.signal.aborted) {
      throw createAbortError('Operation cancelled by AbortSignal');
    }
  }

  /**
   * Helper for broad `catch` blocks so cancellation isn't accidentally swallowed.
   *
   * Example:
   * ```ts
   * try { ... } catch (e) { ctx.rethrowIfCancelled(e); ... }
   * ```
   */
  rethrowIfCancelled(err: unknown): void {
    rethrowIfAborted(err);
  }

  async cancel() {
    await this.v1.runs.cancel({
      ids: [this.action.taskRunExternalId],
    });

    // optimistically abort the run
    this.controller.abort();
  }

  /**
   * Retrieves the output of a parent task.
   * @param parentTask - The a CreateTaskOpts or string of the parent task name.
   * @returns The output of the specified parent task.
   * @throws An error if the task output is not found.
   */
  async parentOutput<L extends OutputType>(
    parentTask: CreateWorkflowTaskOpts<any, L> | CreateWorkflowDurableTaskOpts<any, L> | string
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
   * @hidden
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
   * Gets the payload from the filter that matched when triggering the event.
   * @returns The payload.
   */
  filterPayload(): Record<string, any> {
    return this.data.triggers?.filter_payload || {};
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
    return this.action.taskName;
  }

  /**
   * Gets the ID of the current workflow run.
   * @returns The workflow run ID.
   */
  workflowRunId(): string {
    return this.action.workflowRunId;
  }

  /**
   * Gets the workflow ID of the currently running workflow.
   * @returns The workflow id.
   */
  workflowId(): string | undefined {
    return this.action.workflowId;
  }

  /**
   * Gets the workflow version ID of the currently running workflow.
   * @returns The workflow version ID.
   */
  workflowVersionId(): string | undefined {
    return this.action.workflowVersionId;
  }

  /**
   * Gets the ID of the current task run.
   * @returns The task run ID.
   */
  taskRunExternalId(): string {
    return this.action.taskRunExternalId;
  }

  /**
   * Gets the ID of the current task run.
   * @returns The task run ID.
   * @deprecated use taskRunExternalId() instead
   * @hidden
   */
  taskRunId(): string {
    return this.taskRunExternalId();
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
   * @deprecated use ctx.logger.infoger.info, ctx.logger.infoger.debug, ctx.logger.infoger.warn, ctx.logger.infoger.error, ctx.logger.infoger.trace instead
   * @hidden
   */
  log(message: string, level?: LogLevel, extra?: LogExtra) {
    const { taskRunExternalId } = this.action;

    if (!taskRunExternalId) {
      // log a warning
      this._logger.warn('cannot log from context without stepRunId');
      return Promise.resolve();
    }

    const logger = this.v1.config.logger('ctx', this.v1.config.log_level);
    const contextExtra = {
      workflowRunId: this.action.workflowRunId,
      taskRunExternalId: this.action.taskRunExternalId,
      retryCount: this.action.retryCount,
      workflowName: this.action.jobName,
      ...extra?.extra,
    };

    const promises = [];

    if (!level || level === 'INFO') {
      promises.push(logger.info(message, contextExtra));
    } else if (level === 'DEBUG') {
      promises.push(logger.debug(message, contextExtra));
    } else if (level === 'WARN') {
      promises.push(logger.warn(message, extra?.error, contextExtra));
    } else if (level === 'ERROR') {
      promises.push(logger.error(message, extra?.error, contextExtra));
    }

    // FIXME: this is a hack to get around the fact that the log level is not typed
    promises.push(
      this.v1.event.putLog(
        taskRunExternalId,
        message,
        level as any,
        this.retryCount(),
        extra?.extra
      )
    );

    return Promise.all(promises);
  }

  get logger() {
    return {
      info: (message: string, extra?: any) => {
        return this.log(message, 'INFO', { extra });
      },
      debug: (message: string, extra?: any) => {
        return this.log(message, 'DEBUG', { extra });
      },
      warn: (message: string, extra?: LogExtra) => {
        return this.log(message, 'WARN', extra);
      },
      error: (message: string, extra?: LogExtra) => {
        return this.log(message, 'ERROR', extra);
      },
      util: (key: string, message: string, extra?: LogExtra) => {
        const logger = this.v1.config.logger('ctx', this.v1.config.log_level);
        if (!logger.util) {
          return Promise.resolve();
        }
        return logger.util(key, message, extra?.extra);
      },
    };
  }

  /**
   * Refreshes the timeout for the current task.
   * @param incrementBy - The interval by which to increment the timeout.
   * The interval should be specified in the format of '10s' for 10 seconds, '1m' for 1 minute, or '1d' for 1 day.
   */
  async refreshTimeout(incrementBy: Duration) {
    const { taskRunExternalId } = this.action;

    if (!taskRunExternalId) {
      // log a warning
      this._logger.warn('cannot refresh timeout from context without stepRunId');
      return;
    }

    await this.v1.dispatcher.refreshTimeout(durationToString(incrementBy), taskRunExternalId);
  }

  /**
   * Releases a worker slot for a task run such that the worker can pick up another task.
   * Note: this is an advanced feature that may lead to unexpected behavior if used incorrectly.
   * @returns A promise that resolves when the slot has been released.
   */
  async releaseSlot(): Promise<void> {
    await this.v1.dispatcher.client.releaseSlot({
      taskRunExternalId: this.action.taskRunExternalId,
    });
  }

  /**
   * Streams data from the current task run.
   * @param data - The data to stream (string or binary).
   * @returns A promise that resolves when the data has been streamed.
   */
  async putStream(data: string | Uint8Array) {
    const { taskRunExternalId } = this.action;

    if (!taskRunExternalId) {
      // log a warning
      this._logger.warn('cannot log from context without stepRunId');
      return;
    }

    const index = this._incrementStreamIndex();

    await this.v1.events.putStream(taskRunExternalId, data, index);
  }

  protected spawnOptions(workflow: string | WorkflowV1<any, any>, options?: ChildRunOpts) {
    this.throwIfCancelled();

    let workflowName: string;

    if (typeof workflow === 'string') {
      workflowName = workflow;
    } else {
      workflowName = workflow.name;
    }

    const opts = options || {};
    const { sticky } = opts;

    if (sticky && !this.worker.hasWorkflow(workflowName)) {
      throw new HatchetError(
        `Cannot run with sticky: workflow ${workflowName} is not registered on the worker`
      );
    }

    const { workflowRunId, taskRunExternalId } = this.action;

    const childIndex = this.nextChildIndex();

    const finalOpts = {
      ...opts,
      parentId: workflowRunId,
      parentTaskRunExternalId: taskRunExternalId,
      childIndex,
      childKey: options?.key,
      desiredWorkerId: sticky ? this.worker.id() : undefined,
      _standaloneTaskName:
        workflow instanceof TaskWorkflowDeclaration ? workflow._standalone_task_name : undefined,
    };

    return { workflowName, opts: finalOpts };
  }

  private spawn<Q extends JsonObject, P extends JsonObject>(
    workflow: string | WorkflowV1<Q, P>,
    input: Q,
    options?: ChildRunOpts
  ) {
    const { workflowName, opts } = this.spawnOptions(workflow, options);
    return this.v1.admin.runWorkflow<Q, P>(workflowName, input, opts);
  }

  private spawnBulk<Q extends JsonObject, P extends JsonObject>(
    children: Array<{
      workflow: string | WorkflowV1<Q, P>;
      input: Q;
      options?: ChildRunOpts;
    }>
  ) {
    this.throwIfCancelled();
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
      workflow: string | WorkflowV1<Q, P>;
      input: Q;
      options?: ChildRunOpts;
    }>
  ): Promise<WorkflowRunRef<P>[]> {
    const refs = await this.spawnBulk<Q, P>(children);
    refs.forEach((ref) => {
      ref.defaultSignal = this.abortController.signal;
    });
    return refs;
  }

  /**
   * Runs multiple children workflows in parallel and waits for all results.
   * @param children - An array of objects containing the workflow name, input data, and options for each workflow.
   * @returns A list of results from the children workflows.
   */
  async bulkRunChildren<Q extends JsonObject = any, P extends JsonObject = any>(
    children: Array<{
      workflow: string | WorkflowV1<Q, P>;
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
    workflow: string | WorkflowV1<Q, P> | TaskWorkflowDeclaration<Q, P>,
    input: Q,
    options?: ChildRunOpts
  ): Promise<P> {
    const run = await this.spawn(workflow, input, options);
    // Ensure waiting for the child result aborts when this task is cancelled.

    run.defaultSignal = this.abortController.signal;
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
    workflow: string | WorkflowV1<Q, P>,
    input: Q,
    options?: ChildRunOpts
  ): Promise<WorkflowRunRef<P>> {
    const ref = await this.spawn(workflow, input, options);
    ref.defaultSignal = this.abortController.signal;
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

  /**
   * Gets the external ID of the event that triggered this workflow run, if any.
   * @returns The triggering event external ID, or undefined if the workflow was not triggered by an event.
   */
  triggeringEventId(): string | undefined {
    return this.action.triggeringEventExternalId;
  }

  /**
   * Gets the key of the event that triggered this workflow run, if any.
   * @returns The triggering event key, or undefined if the workflow was not triggered by an event.
   */
  triggeringEventKey(): string | undefined {
    return this.action.triggeringEventKey;
  }
  // FIXME: drop these at some point soon

  /**
   * Get the output of a task.
   * @param task - The name of the task to get the output for.
   * @returns The output of the task.
   * @throws An error if the task output is not found.
   * @deprecated use ctx.parentOutput instead
   * @hidden
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
   * @hidden
   */
  workflowInput(): T {
    return this.input;
  }

  /**
   * Gets the name of the current task.
   * @returns The name of the task.
   * @deprecated use ctx.taskName instead
   * @hidden
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
   * @hidden
   */
  async spawnWorkflows<Q extends JsonObject = any, P extends JsonObject = any>(
    workflows: Array<{
      workflow: string | WorkflowV1<Q, P>;
      input: Q;
      options?: ChildRunOpts;
    }>
  ): Promise<WorkflowRunRef<P>[]> {
    this.throwIfCancelled();
    const { workflowRunId, taskRunExternalId } = this.action;

    const workflowRuns = workflows.map(({ workflow, input, options }) => {
      let workflowName: string;

      if (typeof workflow === 'string') {
        workflowName = workflow;
      } else {
        workflowName = workflow.name;
      }

      const name = applyNamespace(workflowName, this.v1.config.namespace).toLowerCase();

      const opts = options || {};
      const { sticky } = opts;

      if (sticky && !this.worker.hasWorkflow(name)) {
        throw new HatchetError(
          `Cannot run with sticky: workflow ${name} is not registered on the worker`
        );
      }

      // `signal` must never be sent over the wire.
      const optsWithoutSignal: Omit<ChildRunOpts, 'signal'> & { signal?: never } = { ...opts };

      delete (optsWithoutSignal as any).signal;

      const childIndex = this.nextChildIndex();

      const resp = {
        workflowName: name,
        input,
        options: {
          ...optsWithoutSignal,
          parentId: workflowRunId,
          parentTaskRunExternalId: taskRunExternalId,
          childIndex,
          desiredWorkerId: sticky ? this.worker.id() : undefined,
        },
      };
      return resp;
    });

    try {
      const batchSize = 10;

      let resp: WorkflowRunRef<P>[] = [];
      for (let i = 0; i < workflowRuns.length; i += batchSize) {
        const batch = workflowRuns.slice(i, i + batchSize);
        const batchResp = await this.v1.admin.runWorkflows<Q, P>(batch);
        resp = resp.concat(batchResp);
      }

      const res: WorkflowRunRef<P>[] = [];
      resp.forEach((ref, index) => {
        const wf = workflows[index].workflow;
        if (wf instanceof TaskWorkflowDeclaration) {
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
   * @hidden
   */
  async spawnWorkflow<Q extends JsonObject, P extends JsonObject>(
    workflow: string | WorkflowV1<Q, P> | TaskWorkflowDeclaration<Q, P>,
    input: Q,
    options?: ChildRunOpts
  ): Promise<WorkflowRunRef<P>> {
    this.throwIfCancelled();
    const { workflowRunId, taskRunExternalId } = this.action;

    const workflowName = typeof workflow === 'string' ? workflow : workflow.name;

    const name = applyNamespace(workflowName, this.v1.config.namespace).toLowerCase();

    const opts = options || {};
    const { sticky } = opts;

    if (sticky && !this.worker.hasWorkflow(name)) {
      throw new HatchetError(
        `cannot run with sticky: workflow ${name} is not registered on the worker`
      );
    }

    try {
      const childIndex = this.nextChildIndex();

      const resp = await this.v1.admin.runWorkflow<Q, P>(name, input, {
        parentId: workflowRunId,
        parentTaskRunExternalId: taskRunExternalId,
        childIndex,
        desiredWorkerId: sticky ? this.worker.id() : undefined,
        ...opts,
      });

      if (workflow instanceof TaskWorkflowDeclaration) {
        resp._standaloneTaskName = workflow._standalone_task_name;
      }

      return resp;
    } catch (e: any) {
      throw new HatchetError(e.message);
    }
  }

  _incrementStreamIndex() {
    const index = this.streamIndex;
    this.streamIndex += 1;

    return index;
  }
}

/**
 * DurableContext provides helper methods and useful data to durable tasks at runtime.
 * It extends the Context class and includes additional methods for durable execution like sleepFor and waitFor.
 */
export class DurableContext<T, K = {}> extends Context<T, K> {
  private _durableListener: DurableListenerClient;
  private _evictionManager: DurableEvictionManager | undefined;
  private _engineVersion: string | undefined;
  private _waitKey: number = 0;

  constructor(
    action: Action,
    v1: HatchetClient,
    worker: InternalWorker,
    durableListener: DurableListenerClient,
    evictionManager?: DurableEvictionManager,
    engineVersion?: string
  ) {
    super(action, v1, worker);
    this._durableListener = durableListener;
    this._evictionManager = evictionManager;
    this._engineVersion = engineVersion;
  }

  get supportsEviction(): boolean {
    return supportsEviction(this._engineVersion);
  }

  get durableListener(): DurableListenerClient {
    return this._durableListener;
  }

  /**
   * The invocation count for the current durable task. Used for deduplication across replays.
   */
  get invocationCount(): number {
    return this.action.durableTaskInvocationCount ?? 1;
  }

  private get _actionKey(): ActionKey {
    return this.action.key;
  }

  private async withEvictionWait<R>(
    waitKind: string,
    resourceId: string,
    fn: () => Promise<R>
  ): Promise<R> {
    this._evictionManager?.markWaiting(this._actionKey, waitKind, resourceId);
    try {
      return await fn();
    } finally {
      this._evictionManager?.markActive(this._actionKey);
    }
  }

  /**
   * Pauses execution for the specified duration.
   * Duration is "global" meaning it will wait in real time regardless of transient failures like worker restarts.
   * @param duration - The duration to sleep for.
   * @param readableDataKeyOrOptions - Optional readable key string or options object containing readableDataKey and label.
   * @param label - Optional wait label used when readableDataKey is passed as the second argument.
   * @returns A promise that resolves with a SleepResult when the sleep duration has elapsed.
   */
  async sleepFor(duration: Duration, options?: SleepForOptions): Promise<SleepResult>;
  async sleepFor(duration: Duration, readableDataKey?: string): Promise<SleepResult>;
  async sleepFor(
    duration: Duration,
    readableDataKey?: string,
    label?: string
  ): Promise<SleepResult>;
  async sleepFor(
    duration: Duration,
    readableDataKeyOrOptions?: string | SleepForOptions,
    label?: string
  ): Promise<SleepResult> {
    const opts: SleepForOptions =
      typeof readableDataKeyOrOptions === 'string'
        ? { readableDataKey: readableDataKeyOrOptions, label }
        : (readableDataKeyOrOptions ?? {});

    const res = await this.waitFor(
      { sleepFor: duration, readableDataKey: opts.readableDataKey },
      opts.label
    );

    const matches: Record<string, any[]> = res['CREATE'] || {};
    const [firstMatch] = Object.values(matches);

    if (!firstMatch || firstMatch.length === 0) {
      return { durationMs: durationToMs(duration) };
    }

    const [sleep] = firstMatch;
    const sleepDuration: string | undefined = sleep?.sleep_duration;

    if (sleepDuration) {
      const DURATION_RE = /^(?:(\d+)h)?(?:(\d+)m)?(?:(\d+)s)?$/;
      const match = sleepDuration.match(DURATION_RE);
      if (match) {
        const [, h, m, s] = match;
        const ms =
          (parseInt(h ?? '0', 10) * 3600 + parseInt(m ?? '0', 10) * 60 + parseInt(s ?? '0', 10)) *
          1000;
        return { durationMs: ms };
      }
    }

    return { durationMs: durationToMs(duration) };
  }

  /**
   * Pauses execution until the specified conditions are met.
   * Conditions are "global" meaning they will wait in real time regardless of transient failures like worker restarts.
   * @param conditions - The conditions to wait for.
   * @returns A promise that resolves with the event that satisfied the conditions.
   */
  async waitFor(
    conditions: Conditions | Conditions[],
    label?: string
  ): Promise<Record<string, any>> {
    this.throwIfCancelled();

    if (!this.supportsEviction) {
      return this._waitForPreEviction(conditions);
    }

    const rendered = Render(ConditionAction.CREATE, conditions);
    const pbConditions = conditionsToPb(rendered, this.v1.config.namespace);

    const ack = await this._durableListener.sendEvent(
      this.action.taskRunExternalId,
      this.invocationCount,
      {
        kind: 'waitFor',
        waitForConditions: {
          sleepConditions: pbConditions.sleepConditions,
          userEventConditions: pbConditions.userEventConditions,
        },
        label,
      }
    );

    const resourceId =
      rendered
        .map((c) => c.base.readableDataKey)
        .filter(Boolean)
        .join(',') || `node:${ack.nodeId}`;

    return this.withEvictionWait('waitFor', resourceId, async () => {
      const result = await this._durableListener.waitForCallback(
        this.action.taskRunExternalId,
        this.invocationCount,
        ack.branchId,
        ack.nodeId,
        { signal: this.abortController.signal }
      );
      return result.payload || {};
    });
  }

  /**
   * Lightweight wrapper for waiting for a user event. Allows for shorthand usage of
   * `ctx.waitFor` when specifying a user event condition.
   *
   * For more complicated conditions, use `ctx.waitFor` directly.
   *
   * @param key - The event key to wait for.
   * @param expression - An optional CEL expression to filter events.
   * @param payloadSchema - An optional Zod schema to validate and parse the event payload.
   * @returns The event payload, validated against the schema if provided.
   */
  async waitForEvent(
    key: string,
    expression?: string,
    payloadSchema?: undefined,
    scope?: string,
    lookbackWindow?: Duration,
    label?: string
  ): Promise<Record<string, any>>;
  async waitForEvent<T extends z.ZodTypeAny>(
    key: string,
    expression?: string,
    payloadSchema?: T,
    scope?: string,
    lookbackWindow?: Duration,
    label?: string
  ): Promise<z.infer<T>>;
  async waitForEvent(
    key: string,
    expression?: string,
    payloadSchema?: z.ZodTypeAny,
    scope?: string,
    lookbackWindow?: Duration,
    label?: string
  ): Promise<unknown> {
    const now = await this.now();
    const considerEventsSince = lookbackWindow
      ? new Date(now.getTime() - durationToMs(lookbackWindow)).toISOString()
      : undefined;

    const res = await this.waitFor(
      {
        eventKey: key,
        expression,
        scope,
        considerEventsSince,
      },
      label
    );

    // The engine returns an object like:
    // {"CREATE": {"signal_key_1": [{"id": ..., "data": {...}}]}}
    // Since we have a single match, the list will only have one item.
    const matches: Record<string, any[]> = res['CREATE'] || {};
    const [firstMatch] = Object.values(matches);

    if (!firstMatch || firstMatch.length === 0) {
      if (payloadSchema) {
        return payloadSchema.parse({});
      }
      return {};
    }

    const [rawPayload] = firstMatch;

    if (payloadSchema) {
      return payloadSchema.parse(rawPayload);
    }

    return rawPayload;
  }

  /**
   * Durably sleep until a specific timestamp.
   * Uses the memoized `now()` to compute the remaining duration, then delegates to `sleepFor`.
   *
   * @param wakeAt - The timestamp to sleep until.
   * @returns A SleepResult containing the actual duration slept.
   */
  async sleepUntil(wakeAt: Date): Promise<SleepResult> {
    const now = await this.now();
    const remainingMs = wakeAt.getTime() - now.getTime();
    return this.sleepFor(`${Math.max(0, Math.ceil(remainingMs / 1000))}s`);
  }

  /**
   * Get the current timestamp, memoized across replays. Returns the same Date on every replay of the same task run.
   * @returns The memoized current timestamp.
   */
  async now(): Promise<Date> {
    const result = await this.memo(async () => {
      return { ts: new Date().toISOString() };
    }, ['now']);
    return new Date(result.ts);
  }

  private async _waitForPreEviction(
    conditions: Conditions | Conditions[]
  ): Promise<Record<string, any>> {
    const { result, nextWaitKey } = await waitForPreEviction(
      this._durableListener,
      this.action.taskRunExternalId,
      this._waitKey,
      conditions,
      this.v1.config.namespace,
      this.abortController.signal
    );
    this._waitKey = nextWaitKey;
    return result;
  }

  private _buildTriggerOpts<Q extends JsonObject>(
    workflow: string | WorkflowV1<Q, any> | TaskWorkflowDeclaration<Q, any>,
    input?: Q,
    options?: ChildRunOpts
  ) {
    let workflowName: string;
    if (typeof workflow === 'string') {
      workflowName = workflow;
    } else {
      workflowName = workflow.name;
    }

    workflowName = applyNamespace(workflowName, this.v1.config.namespace).toLowerCase();

    const childIndex = this.nextChildIndex();

    const triggerOpts = {
      name: workflowName,
      input: JSON.stringify(input || {}),
      parentId: this.action.workflowRunId,
      parentTaskRunExternalId: this.action.taskRunExternalId,
      childIndex,
      childKey: options?.key,
      additionalMetadata: options?.additionalMetadata
        ? JSON.stringify(options.additionalMetadata)
        : undefined,
      desiredWorkerId: options?.sticky ? this.worker.id() : undefined,
      priority: options?.priority,
      desiredWorkerLabels: {},
    };

    return { workflowName, triggerOpts };
  }

  /**
   * Spawns a child workflow through the durable event log, waits for the child to complete.
   * @param workflow - The workflow to spawn.
   * @param input - The input data for the child workflow.
   * @param options - Options for spawning the child workflow.
   * @returns The result of the child workflow.
   */
  async spawnChild<Q extends JsonObject, P extends OutputType>(
    workflow: string | WorkflowV1<Q, P> | TaskWorkflowDeclaration<Q, P>,
    input?: Q,
    options?: ChildRunOpts
  ): Promise<P> {
    if (!this.supportsEviction) {
      const { workflowName, opts } = this.spawnOptions(workflow, options);
      const ref = await this.v1.admin.runWorkflow(workflowName, (input || {}) as Q, opts);
      ref.defaultSignal = this.abortController.signal;
      return ref.output as Promise<P>;
    }

    const results = await this.spawnChildren<Q, P>([
      { workflow, input: (input || {}) as Q, options },
    ]);
    return results[0];
  }

  /**
   * Spawns multiple child workflows through the durable event log, waits for all to complete.
   * @param children - An array of objects containing the workflow, input, and options for each child.
   * @returns A list of results from the child workflows.
   */
  async spawnChildren<Q extends JsonObject, P extends OutputType>(
    children: Array<{
      workflow: string | WorkflowV1<Q, P> | TaskWorkflowDeclaration<Q, P>;
      input: Q;
      options?: ChildRunOpts;
    }>
  ): Promise<P[]> {
    this.throwIfCancelled();

    if (!this.supportsEviction) {
      const workflows = children.map((c) => {
        const { workflowName, opts } = this.spawnOptions(c.workflow, c.options);
        return { workflowName, input: c.input, options: opts };
      });
      const refs = await this.v1.admin.runWorkflows(workflows);
      for (const r of refs) {
        r.defaultSignal = this.abortController.signal;
      }
      return Promise.all(refs.map((r) => r.output)) as Promise<P[]>;
    }

    const triggerOptsList = children.map((child) => {
      const { triggerOpts } = this._buildTriggerOpts(child.workflow, child.input, child.options);
      return triggerOpts;
    });

    const ack = await this._durableListener.sendEvent(
      this.action.taskRunExternalId,
      this.invocationCount,
      {
        kind: 'runChildren',
        triggerOpts: triggerOptsList,
      }
    );

    const results = await Promise.all(
      ack.runEntries.map((entry) =>
        this.withEvictionWait('runChild', `workflow:bulk-child`, async () => {
          const result = await this._durableListener.waitForCallback(
            this.action.taskRunExternalId,
            this.invocationCount,
            entry.branchId,
            entry.nodeId,
            { signal: this.abortController.signal }
          );
          return (result.payload || {}) as P;
        })
      )
    );

    return results;
  }

  /**
   * Memoize a function by storing its result in durable storage. Avoids recomputation on replay.
   *
   * @param fn - The async function to compute the value.
   * @param deps - Dependency values that form the memoization key.
   * @returns The memoized value, either from durable storage or freshly computed.
   */
  private async memo<R>(fn: () => Promise<R>, deps: readonly unknown[]): Promise<R> {
    this.throwIfCancelled();

    if (!this.supportsEviction) {
      return fn();
    }

    const memoKey = computeMemoKey(this.action.taskRunExternalId, deps);

    const ack = await this._durableListener.sendEvent(
      this.action.taskRunExternalId,
      this.invocationCount,
      {
        kind: 'memo',
        memoKey,
      }
    );

    if (ack.memoAlreadyExisted && ack.memoResultPayload && ack.memoResultPayload.length > 0) {
      const serialized = new TextDecoder().decode(ack.memoResultPayload);
      return JSON.parse(serialized) as R;
    }

    const result = await fn();
    const serializedResult = new TextEncoder().encode(JSON.stringify(result));

    await this._durableListener.sendMemoCompletedNotification(
      this.action.taskRunExternalId,
      ack.nodeId,
      ack.branchId,
      this.invocationCount,
      memoKey,
      serializedResult
    );

    return result;
  }
}

function computeMemoKey(taskRunExternalId: string, args: readonly unknown[]): Uint8Array {
  const h = createHash('sha256');
  h.update(taskRunExternalId);
  h.update(JSON.stringify(args));
  return new Uint8Array(h.digest());
}
