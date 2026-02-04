/* eslint-disable no-underscore-dangle */
import { WorkerLabels } from '@hatchet/clients/dispatcher/dispatcher-client';
import { LegacyHatchetClient } from '@hatchet/clients/hatchet-client';
import { Workflow as V0Workflow } from '@hatchet/workflow';
import { WebhookWorkerCreateRequest } from '@hatchet/clients/rest/generated/data-contracts';
import { BaseWorkflowDeclaration } from '../../declaration';
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

  /** Internal reference to the underlying V0 worker implementation */
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
    const resolvedOptions = resolveWorkerOptions(options);
    const opts = {
      name,
      ...resolvedOptions,
    };

    const internalWorker = new V1Worker(v1, opts);
    const worker = new Worker(v1, v0, internalWorker, options, name);
    await worker.registerWorkflows(options.workflows);
    return worker;
  }

  /**
   * Registers workflows with the worker
   * @param workflows - Array of workflows to register
   * @returns Array of registered workflow promises
   */
  async registerWorkflows(workflows?: Array<BaseWorkflowDeclaration<any, any> | V0Workflow>) {
    for (const wf of workflows || []) {
      if (wf instanceof BaseWorkflowDeclaration) {
        // TODO check if tenant is V1
        await this._internal.registerWorkflowV1(wf);

        if (wf.definition._durableTasks.length > 0) {
          this._internal.registerDurableActionsV1(wf.definition);
        }
      } else {
        // fallback to v0 client for backwards compatibility
        await this._internal.registerWorkflow(wf);
      }
    }
  }

  /**
   * Registers a single workflow with the worker
   * @param workflow - The workflow to register
   * @returns A promise that resolves when the workflow is registered
   * @deprecated use registerWorkflows instead
   */
  registerWorkflow(workflow: BaseWorkflowDeclaration<any, any> | V0Workflow) {
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

  /**
   * Register a webhook with the worker
   * @param webhook - The webhook to register
   * @returns A promise that resolves when the webhook is registered
   */
  registerWebhook(webhook: WebhookWorkerCreateRequest) {
    return this._internal.registerWebhook(webhook);
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
