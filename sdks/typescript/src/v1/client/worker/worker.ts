/* eslint-disable no-underscore-dangle */
import { WorkerLabels } from '@hatchet/clients/dispatcher/dispatcher-client';
import { LegacyHatchetClient } from '@hatchet/clients/hatchet-client';
import { BaseWorkflowDeclaration } from '../../declaration';
import type { LegacyWorkflow } from '../../../legacy/legacy-transformer';
import { normalizeWorkflows } from '../../../legacy/legacy-transformer';
import { HatchetClient } from '../..';
import { V1Worker } from './worker-internal';
import { resolveWorkerOptions, type WorkerSlotOptions } from './slot-utils';

/**
 * Options for creating a new hatchet worker
 * @interface CreateWorkerOpts
 */
export interface CreateWorkerOpts extends WorkerSlotOptions {
  /** (optional) Worker labels for affinity-based assignment */
  labels?: WorkerLabels;
  /** (optional) Whether to handle kill signals */
  handleKill?: boolean;
}

/**
 * HatchetWorker class for workflow execution runtime
 */
export class Worker {
  config: CreateWorkerOpts;
  name: string;
  _v1: HatchetClient;
  _v0: LegacyHatchetClient;

  _internal: V1Worker;

  /**
   * Creates a new HatchetWorker instance
   * @param nonDurable - The V0 worker implementation
   */
  constructor(
    v1: HatchetClient,
    v0: LegacyHatchetClient,
    nonDurable: V1Worker,
    config: CreateWorkerOpts,
    name: string
  ) {
    this._v1 = v1;
    this._v0 = v0;
    this._internal = nonDurable;
    this.config = config;
    this.name = name;
  }

  /**
   * Creates and initializes a new HatchetWorker
   * @param v0 - The HatchetClient instance
   * @param options - Worker creation options
   * @returns A new HatchetWorker instance
   */
  static async create(
    v1: HatchetClient,
    v0: LegacyHatchetClient,
    name: string,
    options: CreateWorkerOpts
  ) {
    // Normalize any legacy workflows before resolving worker options
    const normalizedOptions = {
      ...options,
      workflows: options.workflows ? normalizeWorkflows(options.workflows) : undefined,
    };

    const resolvedOptions = resolveWorkerOptions(normalizedOptions);
    const opts = {
      name,
      ...resolvedOptions,
    };

    const internalWorker = new V1Worker(v1, opts);
    const worker = new Worker(v1, v0, internalWorker, normalizedOptions, name);
    await worker.registerWorkflows(normalizedOptions.workflows);
    return worker;
  }

  /**
   * Registers workflows with the worker.
   * Accepts both v1 BaseWorkflowDeclaration and legacy Workflow objects.
   * Legacy workflows are automatically transformed and a deprecation warning is emitted.
   * @param workflows - Array of workflows to register
   * @returns Array of registered workflow promises
   */
  async registerWorkflows(workflows?: Array<BaseWorkflowDeclaration<any, any> | LegacyWorkflow>) {
    const normalized = workflows ? normalizeWorkflows(workflows) : [];
    for (const wf of normalized) {
      await this._internal.registerWorkflowV1(wf);

      if (wf.definition._durableTasks.length > 0) {
        this._internal.registerDurableActionsV1(wf.definition);
      }
    }
  }

  /**
   * Registers a single workflow with the worker.
   * Accepts both v1 BaseWorkflowDeclaration and legacy Workflow objects.
   * Legacy workflows are automatically transformed and a deprecation warning is emitted.
   * @param workflow - The workflow to register
   * @returns A promise that resolves when the workflow is registered
   * @deprecated use registerWorkflows instead
   */
  registerWorkflow(workflow: BaseWorkflowDeclaration<any, any> | LegacyWorkflow) {
    return this.registerWorkflows([workflow]);
  }

  /**
   * Starts the worker
   * @returns Promise that resolves when the worker is stopped or killed
   */
  start() {
    return this._internal.start();
  }

  /**
   * Stops the worker
   * @returns Promise that resolves when the worker stops
   */
  stop() {
    return this._internal.stop();
  }

  /**
   * Updates or inserts worker labels
   * @param labels - Worker labels to update
   * @returns Promise that resolves when labels are updated
   */
  upsertLabels(labels: WorkerLabels) {
    return this._internal.upsertLabels(labels);
  }

  /**
   * Get the labels for the worker
   * @returns The labels for the worker
   */
  getLabels() {
    return this._internal.labels;
  }

  async isPaused() {
    if (!this._internal?.workerId) {
      return false;
    }

    return this._v1.workers.isPaused(this._internal.workerId);
  }

  // TODO docstrings
  pause() {
    if (!this._internal?.workerId) {
      return Promise.resolve();
    }

    return this._v1.workers.pause(this._internal.workerId);
  }

  unpause() {
    if (!this._internal?.workerId) {
      return Promise.resolve();
    }

    return this._v1.workers.unpause(this._internal.workerId);
  }
}

export { testingExports as __testing } from './slot-utils';
