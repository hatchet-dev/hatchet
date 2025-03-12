import { WorkerLabels } from '@hatchet/clients/dispatcher/dispatcher-client';
import { InternalHatchetClient } from '@hatchet/clients/hatchet-client';
import { Worker as V0Worker } from '@clients/worker';
import { Workflow as V0Workflow } from '@hatchet/workflow';
import { Workflow } from '../workflow';

/**
 * Options for creating a new hatchet worker
 * @interface CreateWorkerOpts
 */
export interface CreateWorkerOpts {
  /** Maximum number of concurrent runs on this worker */
  slots?: number;
  /** Array of workflows to register */
  workflows?: Workflow<any, any>[] | V0Workflow[];
  /** Worker labels for affinity-based assignment */
  labels?: WorkerLabels;
  /** Whether to handle kill signals */
  handleKill?: boolean;
  /** @deprecated Use slots instead */
  maxRuns?: number;
}

/**
 * HatchetWorker class for workflow execution runtime
 */
export class HatchetWorker {
  /** Internal reference to the underlying V0 worker implementation */
  v0: V0Worker;

  /**
   * Creates a new HatchetWorker instance
   * @param v0 - The V0 worker implementation
   */
  constructor(v0: V0Worker) {
    this.v0 = v0;
  }

  /**
   * Creates and initializes a new HatchetWorker
   * @param v0 - The HatchetClient instance
   * @param options - Worker creation options
   * @returns A new HatchetWorker instance
   */
  static async create(v0: InternalHatchetClient, name: string, options: CreateWorkerOpts) {
    const v0worker = await v0.worker(name, {
      ...options,
      maxRuns: options.slots || options.maxRuns,
    });
    const worker = new HatchetWorker(v0worker);
    await worker.registerWorkflows(options.workflows);
    return worker;
  }

  /**
   * Registers workflows with the worker
   * @param workflows - Array of workflows to register
   * @returns Array of registered workflow promises
   */
  registerWorkflows(workflows?: Array<Workflow<any, any> | V0Workflow>) {
    return workflows?.map((wf) => {
      if (wf instanceof Workflow) {
        const { definition } = wf;
        return this.v0.registerWorkflow({
          id: definition.name,
          description: definition.description || '',
          version: definition.version || '',
          sticky: definition.sticky,
          scheduleTimeout: definition.scheduleTimeout,
          on: definition.on,
          concurrency: definition.concurrency,
          steps: definition.tasks.map((task) => ({
            name: task.name,
            parents: task.parents?.map((p) => p.name),
            run: (ctx) => task.fn(ctx.workflowInput(), ctx),
            timeout: task.timeout,
            retries: task.retries,
            rate_limits: task.rateLimits,
            worker_labels: task.workerLabels,
            backoff: task.backoff,
          })),
        });
      }
      // Register v0 workflow
      return this.v0.registerWorkflow(wf);
    });
  }

  /**
   * Registers a single workflow with the worker
   * @param workflow - The workflow to register
   * @returns A promise that resolves when the workflow is registered
   * @deprecated use registerWorkflows instead
   */
  registerWorkflow(workflow: Workflow<any, any> | V0Workflow) {
    return this.registerWorkflows([workflow]);
  }

  /**
   * Starts the worker
   * @returns Promise that resolves when the worker is stopped or killed
   */
  start() {
    return this.v0.start();
  }

  /**
   * Stops the worker
   * @returns Promise that resolves when the worker stops
   */
  stop() {
    return this.v0.stop();
  }

  /**
   * Updates or inserts worker labels
   * @param labels - Worker labels to update
   * @returns Promise that resolves when labels are updated
   */
  upsertLabels(labels: WorkerLabels) {
    return this.v0.upsertLabels(labels);
  }
}
