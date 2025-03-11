import { ClientConfig, HatchetClient, HatchetClientOptions } from '@hatchet/clients/hatchet-client';
import { AxiosRequestConfig } from 'axios';
import WorkflowRunRef from '@hatchet/util/workflow-run-ref';
import { Workflow as V0Workflow } from '@hatchet/workflow';
import { CreateWorkflow, CreateWorkflowOpts, RunOpts, Workflow } from '../workflow';
import { IHatchetClient } from './client.interface';
import { CreateWorkerOpts, HatchetWorker } from './worker';

/**
 * HatchetV1 implements the main client interface for interacting with the Hatchet workflow engine.
 * It provides methods for creating and executing workflows, as well as managing workers.
 */
export class HatchetV1 implements IHatchetClient {
  /** The underlying v0 client instance */
  v0: HatchetClient;

  /**
   * Creates a new HatchetV1 client instance.
   * @param config - Optional configuration for the client
   * @param options - Optional client options
   * @param axiosConfig - Optional Axios configuration for HTTP requests
   */
  constructor(
    config?: Partial<ClientConfig>,
    options?: HatchetClientOptions,
    axiosConfig?: AxiosRequestConfig
  ) {
    this.v0 = new HatchetClient(config, options, axiosConfig);
  }

  /**
   * Static factory method to create a new HatchetV1 client instance.
   * @param config - Optional configuration for the client
   * @param options - Optional client options
   * @param axiosConfig - Optional Axios configuration for HTTP requests
   * @returns A new HatchetV1 client instance
   */
  static init(
    config?: Partial<ClientConfig>,
    options?: HatchetClientOptions,
    axiosConfig?: AxiosRequestConfig
  ): HatchetV1 {
    return new HatchetV1(config, options, axiosConfig);
  }

  /**
   * Creates a new workflow definition.
   * @template T - The input type for the workflow
   * @template K - The return type of the workflow
   * @param options - Configuration options for creating the workflow
   * @returns A new Workflow instance
   * @note It is possible to create an orphaned workflow if no client is available using @hatchet/client CreateWorkflow
   */
  createWorkflow<T = any, K = any>(options: CreateWorkflowOpts): Workflow<T, K> {
    return CreateWorkflow<T, K>(options, this);
  }

  /**
   * Triggers a workflow run without waiting for completion.
   * @template T - The input type for the workflow
   * @template K - The return type of the workflow
   * @param workflow - The workflow to run, either as a Workflow instance or workflow name
   * @param input - The input data for the workflow
   * @param options - Configuration options for the workflow run
   * @returns A WorkflowRunRef containing the run ID and methods to interact with the run
   */
  enqueue<T = any, K = any>(
    workflow: Workflow<T, K> | string | V0Workflow,
    input: T,
    options: RunOpts
  ): WorkflowRunRef<K> {
    let name: string;
    if (typeof workflow === 'string') {
      name = workflow;
    } else if ('id' in workflow) {
      name = workflow.id;
    } else if ('definition' in workflow) {
      name = workflow.definition.name;
    } else {
      throw new Error('unable to identify workflow');
    }

    return this.v0.admin.runWorkflow<T, K>(name, input, options);
  }

  /**
   * Triggers a workflow run and waits for the result.
   * @template T - The input type for the workflow
   * @template K - The return type of the workflow
   * @param workflow - The workflow to run, either as a Workflow instance or workflow name
   * @param input - The input data for the workflow
   * @param options - Configuration options for the workflow run
   * @returns A promise that resolves with the workflow result
   */
  async run<T = any, K = any>(
    workflow: Workflow<T, K> | string | V0Workflow,
    input: T,
    options: RunOpts = {}
  ): Promise<K> {
    const run = this.enqueue<T, K>(workflow, input, options);
    return run.result() as Promise<K>;
  }

  /**
   * Creates a new worker instance for processing workflow tasks.
   * @param options - Configuration options for creating the worker
   * @returns A promise that resolves with a new HatchetWorker instance
   */
  createWorker(options?: CreateWorkerOpts): Promise<HatchetWorker> {
    const defaults: CreateWorkerOpts = {
      name: process.env.HOSTNAME,
    };

    return HatchetWorker.create(this.v0, {
      ...defaults,
      ...options,
    });
  }
}
