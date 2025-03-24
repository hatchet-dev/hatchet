import { WorkerLabels } from '@hatchet/clients/dispatcher/dispatcher-client';
import { InternalHatchetClient } from '@hatchet/clients/hatchet-client';
import { V0Worker } from '@clients/worker';
import { Workflow as V0Workflow } from '@hatchet/workflow';
import { WebhookWorkerCreateRequest } from '@hatchet/clients/rest/generated/data-contracts';
import { WorkflowDeclaration } from '../workflow';
import { HatchetClient } from '..';

const DEFAULT_DURABLE_SLOTS = 1_000;

/**
 * Options for creating a new hatchet worker
 * @interface CreateWorkerOpts
 */
export interface CreateWorkerOpts {
  /** (optional) Maximum number of concurrent runs on this worker, defaults to 100 */
  slots?: number;
  /** (optional) Array of workflows to register */
  workflows?: WorkflowDeclaration<any, any>[] | V0Workflow[];
  /** (optional) Worker labels for affinity-based assignment */
  labels?: WorkerLabels;
  /** (optional) Whether to handle kill signals */
  handleKill?: boolean;
  /** @deprecated Use slots instead */
  maxRuns?: number;

  /** (optional) Maximum number of concurrent runs on the durable worker, defaults to 1,000 */
  durableSlots?: number;
}

/**
 * HatchetWorker class for workflow execution runtime
 */
export class Worker {
  config: CreateWorkerOpts;
  name: string;
  v1: HatchetClient;
  v0: InternalHatchetClient;

  /** Internal reference to the underlying V0 worker implementation */
  nonDurable: V0Worker;
  durable?: V0Worker;

  /**
   * Creates a new HatchetWorker instance
   * @param nonDurable - The V0 worker implementation
   */
  constructor(
    v1: HatchetClient,
    v0: InternalHatchetClient,
    nonDurable: V0Worker,
    config: CreateWorkerOpts,
    name: string
  ) {
    this.v1 = v1;
    this.v0 = v0;
    this.nonDurable = nonDurable;
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
    v0: InternalHatchetClient,
    name: string,
    options: CreateWorkerOpts
  ) {
    const v0worker = await v0.worker(name, {
      ...options,
      maxRuns: options.slots || options.maxRuns,
    });
    const worker = new Worker(v1, v0, v0worker, options, name);
    await worker.registerWorkflows(options.workflows);
    return worker;
  }

  /**
   * Registers workflows with the worker
   * @param workflows - Array of workflows to register
   * @returns Array of registered workflow promises
   */
  async registerWorkflows(workflows?: Array<WorkflowDeclaration<any, any> | V0Workflow>) {
    return Promise.all(
      workflows?.map(async (wf) => {
        if (wf instanceof WorkflowDeclaration) {
          // TODO check if tenant is V1
          const register = this.nonDurable.registerWorkflowV1(wf);

          if (wf.definition.durableTasks.length > 0) {
            if (!this.durable) {
              this.durable = await this.v0.worker(`${this.name}-durable`, {
                ...this.config,
                maxRuns: this.config.durableSlots || DEFAULT_DURABLE_SLOTS,
              });
            }
            this.durable.registerDurableActionsV1(wf.definition);
          }

          return register;
        }

        // fallback to v0 client for backwards compatibility
        return this.nonDurable.registerWorkflow(wf);
      }) || []
    );
  }

  /**
   * Registers a single workflow with the worker
   * @param workflow - The workflow to register
   * @returns A promise that resolves when the workflow is registered
   * @deprecated use registerWorkflows instead
   */
  registerWorkflow(workflow: WorkflowDeclaration<any, any> | V0Workflow) {
    return this.registerWorkflows([workflow]);
  }

  /**
   * Starts the worker
   * @returns Promise that resolves when the worker is stopped or killed
   */
  start() {
    const workers = [this.nonDurable];

    if (this.durable) {
      workers.push(this.durable);
    }

    return Promise.all(workers.map((w) => w.start()));
  }

  /**
   * Stops the worker
   * @returns Promise that resolves when the worker stops
   */
  stop() {
    const workers = [this.nonDurable];

    if (this.durable) {
      workers.push(this.durable);
    }

    return Promise.all(workers.map((w) => w.stop()));
  }

  /**
   * Updates or inserts worker labels
   * @param labels - Worker labels to update
   * @returns Promise that resolves when labels are updated
   */
  upsertLabels(labels: WorkerLabels) {
    return this.nonDurable.upsertLabels(labels);
  }

  /**
   * Get the labels for the worker
   * @returns The labels for the worker
   */
  getLabels() {
    return this.nonDurable.labels;
  }

  /**
   * Register a webhook with the worker
   * @param webhook - The webhook to register
   * @returns A promise that resolves when the webhook is registered
   */
  registerWebhook(webhook: WebhookWorkerCreateRequest) {
    return this.nonDurable.registerWebhook(webhook);
  }

  async isPaused() {
    const promises: Promise<any>[] = [];
    if (this.nonDurable?.workerId) {
      promises.push(this.v1.workers.isPaused(this.nonDurable.workerId));
    }
    if (this.durable?.workerId) {
      promises.push(this.v1.workers.isPaused(this.durable.workerId));
    }

    const res = await Promise.all(promises);

    return !res.includes(false);
  }

  // TODO docstrings
  pause() {
    const promises: Promise<any>[] = [];
    if (this.nonDurable?.workerId) {
      promises.push(this.v1.workers.pause(this.nonDurable.workerId));
    }
    if (this.durable?.workerId) {
      promises.push(this.v1.workers.pause(this.durable.workerId));
    }
    return Promise.all(promises);
  }

  unpause() {
    const promises: Promise<any>[] = [];
    if (this.nonDurable?.workerId) {
      promises.push(this.v1.workers.unpause(this.nonDurable.workerId));
    }
    if (this.durable?.workerId) {
      promises.push(this.v1.workers.unpause(this.durable.workerId));
    }
    return Promise.all(promises);
  }
}
