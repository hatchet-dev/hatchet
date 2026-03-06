/* eslint-disable no-underscore-dangle */
import { WorkerLabels } from '@hatchet/clients/dispatcher/dispatcher-client';
import sleep from '@hatchet/util/sleep';
import { BaseWorkflowDeclaration } from '../../declaration';
import type { LegacyWorkflow } from '../../../legacy/legacy-transformer';
import { normalizeWorkflows } from '../../../legacy/legacy-transformer';
import { HatchetClient } from '../..';
import { InternalWorker } from './worker-internal';
import { resolveWorkerOptions, type WorkerSlotOptions } from './slot-utils';
import {
  isLegacyEngine,
  fetchEngineVersion,
  LegacyDualWorker,
  emitDeprecationNotice,
} from './deprecated';
import { MinEngineVersion, supportsEviction } from './engine-version';

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

  /** Internal reference to the underlying V0 worker implementation */
  _internal: InternalWorker;

  /** Set when connected to a legacy engine that needs dual-worker architecture */
  private _legacyWorker: LegacyDualWorker | undefined;

  /** Tracks all workflows registered after construction (via registerWorkflow/registerWorkflows) */
  private _registeredWorkflows: Array<BaseWorkflowDeclaration<any, any> | LegacyWorkflow> = [];

  /**
   * Creates a new HatchetWorker instance
   * @param nonDurable - The V0 worker implementation
   */
  constructor(
    v1: HatchetClient,
    nonDurable: InternalWorker,
    config: CreateWorkerOpts,
    name: string
  ) {
    this._v1 = v1;
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
  static async create(v1: HatchetClient, name: string, options: CreateWorkerOpts) {
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

    const internalWorker = new InternalWorker(v1, opts);
    const worker = new Worker(v1, internalWorker, normalizedOptions, name);
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
      await this._internal.registerWorkflow(wf);

      if (wf.definition._durableTasks.length > 0) {
        this._internal.registerDurableActions(wf.definition);
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
  async start() {
    // Check engine version and fall back to legacy dual-worker mode if needed
    if (await isLegacyEngine(this._v1)) {
      // Include workflows registered after construction (via registerWorkflow/registerWorkflows)
      // so the legacy worker picks them up.
      const legacyConfig: CreateWorkerOpts = {
        ...this.config,
        workflows: this._registeredWorkflows.length
          ? (this._registeredWorkflows as BaseWorkflowDeclaration<any, any>[])
          : this.config.workflows,
      };
      this._legacyWorker = await LegacyDualWorker.create(this._v1, this.name, legacyConfig);
      return this._legacyWorker.start();
    }

    const engineVersion = await fetchEngineVersion(this._v1).catch(() => undefined);
    this._checkEvictionSupport(engineVersion);
    this._internal.engineVersion = engineVersion;

    return this._internal.start();
  }

  private _checkEvictionSupport(engineVersion: string | undefined): void {
    if (supportsEviction(engineVersion)) return;

    const workflows = (this.config.workflows || []) as BaseWorkflowDeclaration<any, any>[];
    const tasksWithEviction: string[] = [];

    for (const wf of workflows) {
      // eslint-disable-next-line no-continue
      if (!(wf instanceof BaseWorkflowDeclaration)) continue;
      for (const task of wf.definition._durableTasks) {
        if (task.evictionPolicy) {
          tasksWithEviction.push(`${wf.definition.name}:${task.name}`);
        }
      }
    }

    if (tasksWithEviction.length === 0) return;

    const names = tasksWithEviction.join(', ');
    const logger = this._v1.config.logger('Worker', this._v1.config.log_level);
    emitDeprecationNotice(
      'pre-eviction-engine',
      `Engine ${engineVersion || 'unknown'} does not support durable eviction ` +
        `(requires >= ${MinEngineVersion.DURABLE_EVICTION}). ` +
        `Eviction policies will be ignored for tasks: ${names}. ` +
        `Please upgrade your Hatchet engine.`,
      new Date('2026-03-01T00:00:00Z'),
      logger
    );
  }

  /**
   * Stops the worker
   * @returns Promise that resolves when the worker stops
   */
  stop() {
    if (this._legacyWorker) {
      return this._legacyWorker.stop();
    }
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

  /**
   * Waits until the worker has connected and registered with the server.
   * Polls every 200ms. Use after start() to avoid fixed sleeps before running workflows.
   */
  async waitUntilReady(timeoutMs = 10_000): Promise<void> {
    if (this._legacyWorker) {
      await sleep(2000);
      return;
    }
    const pollInterval = 200;
    const start = Date.now();
    while (Date.now() - start < timeoutMs) {
      // start() may asynchronously detect a legacy engine and set _legacyWorker
      // after waitUntilReady has already entered this loop
      if (this._legacyWorker) {
        await sleep(2000);
        return;
      }
      if (this._internal?.workerId) return;
      await sleep(pollInterval);
    }
    throw new Error(`Worker ${this.name} did not become ready within ${timeoutMs}ms`);
  }
}

export { testingExports as __testing } from './slot-utils';
