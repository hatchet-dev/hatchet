/* eslint-disable no-underscore-dangle */
import {
  ClientConfig,
  InternalHatchetClient,
  HatchetClientOptions,
} from '@hatchet/clients/hatchet-client';
import { AxiosRequestConfig } from 'axios';
import WorkflowRunRef from '@hatchet/util/workflow-run-ref';
import { Workflow as V0Workflow } from '@hatchet/workflow';
import { JsonObject } from '@hatchet/step';
import { CreateWorkflow, CreateWorkflowOpts, RunOpts, WorkflowDeclaration } from '../workflow';
import { IHatchetClient } from './client.interface';
import { CreateWorkerOpts, Worker } from './worker';
import { MetricsClient } from './features/metrics';
import { WorkersClient } from './features/workers';
import { WorkflowsClient } from './features/workflows';
import { RunsClient } from './features/runs';

/**
 * HatchetV1 implements the main client interface for interacting with the Hatchet workflow engine.
 * It provides methods for creating and executing workflows, as well as managing workers.
 */
export class HatchetClient implements IHatchetClient {
  /** The underlying v0 client instance */
  v0: InternalHatchetClient;

  /** The tenant ID for the Hatchet client */
  get tenantId() {
    return this.v0.tenantId;
  }

  /**
   * Creates a new Hatchet client instance.
   * @param config - Optional configuration for the client
   * @param options - Optional client options
   * @param axiosConfig - Optional Axios configuration for HTTP requests
   */
  constructor(
    config?: Partial<ClientConfig>,
    options?: HatchetClientOptions,
    axiosConfig?: AxiosRequestConfig
  ) {
    this.v0 = new InternalHatchetClient(config, options, axiosConfig);
  }

  /**
   * Static factory method to create a new Hatchet client instance.
   * @param config - Optional configuration for the client
   * @param options - Optional client options
   * @param axiosConfig - Optional Axios configuration for HTTP requests
   * @returns A new Hatchet client instance
   */
  static init(
    config?: Partial<ClientConfig>,
    options?: HatchetClientOptions,
    axiosConfig?: AxiosRequestConfig
  ): HatchetClient {
    return new HatchetClient(config, options, axiosConfig);
  }

  /**
   * Creates a new workflow definition.
   * @template T - The input type for the workflow
   * @template K - The return type of the workflow
   * @param options - Configuration options for creating the workflow
   * @returns A new Workflow instance
   * @note It is possible to create an orphaned workflow if no client is available using @hatchet/client CreateWorkflow
   */
  workflow<T extends JsonObject = any, K extends JsonObject = any>(
    options: CreateWorkflowOpts
  ): WorkflowDeclaration<T, K> {
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
  runNoWait<T extends JsonObject = any, K extends JsonObject = any>(
    workflow: WorkflowDeclaration<T, K> | string | V0Workflow,
    input: T,
    options: RunOpts
  ): WorkflowRunRef<K> {
    let name: string;
    if (typeof workflow === 'string') {
      name = workflow;
    } else if ('id' in workflow) {
      name = workflow.id;
    } else {
      throw new Error('unable to identify workflow');
    }

    return this.v0.admin.runWorkflow<T, K>(name, input, options);
  }

  /**
   * @alias run
   * Triggers a workflow run and waits for the result.
   * @template T - The input type for the workflow
   * @template K - The return type of the workflow
   * @param workflow - The workflow to run, either as a Workflow instance or workflow name
   * @param input - The input data for the workflow
   * @param options - Configuration options for the workflow run
   * @returns A promise that resolves with the workflow result
   */
  async runAndWait<T extends JsonObject = any, K extends JsonObject = any>(
    workflow: WorkflowDeclaration<T, K> | string | V0Workflow,
    input: T,
    options: RunOpts = {}
  ): Promise<K> {
    return this.run<T, K>(workflow, input, options);
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
  async run<T extends JsonObject = any, K extends JsonObject = any>(
    workflow: WorkflowDeclaration<T, K> | string | V0Workflow,
    input: T,
    options: RunOpts = {}
  ): Promise<K> {
    const run = this.runNoWait<T, K>(workflow, input, options);
    return run.output as Promise<K>;
  }

  /**
   * Get the cron client for creating and managing cron workflow runs
   * @returns A cron client instance
   */
  get crons() {
    return this.v0.cron;
  }

  /**
   * Get the cron client for creating and managing cron workflow runs
   * @returns A cron client instance
   * @deprecated use client.crons instead
   */
  get cron() {
    return this.crons;
  }

  /**
   * Get the schedules client for creating and managing scheduled workflow runs
   * @returns A schedules client instance
   */
  get schedules() {
    return this.v0.schedule;
  }

  /**
   * Get the schedule client for creating and managing scheduled workflow runs
   * @returns A schedule client instance
   * @deprecated use client.schedules instead
   */
  get schedule() {
    return this.schedules;
  }

  /**
   * Get the event client for creating and managing event workflow runs
   * @returns A event client instance
   */
  get events() {
    return this.v0.event;
  }

  /**
   * Get the event client for creating and managing event workflow runs
   * @returns A event client instance
   * @deprecated use client.events instead
   */
  get event() {
    return this.events;
  }

  private _metrics: MetricsClient | undefined;

  /**
   * Get the metrics client for creating and managing metrics
   * @returns A metrics client instance
   */
  get metrics() {
    if (!this._metrics) {
      this._metrics = new MetricsClient(this);
    }
    return this._metrics;
  }

  private _runs: RunsClient | undefined;

  /**
   * Get the runs client for creating and managing runs
   * @returns A runs client instance
   */
  get runs() {
    if (!this._runs) {
      this._runs = new RunsClient(this);
    }
    return this._runs;
  }

  private _workflows: WorkflowsClient | undefined;

  /**
   * Get the workflows client for creating and managing workflows
   * @returns A workflows client instance
   */
  get workflows() {
    if (!this._workflows) {
      this._workflows = new WorkflowsClient(this);
    }
    return this._workflows;
  }

  private _workers: WorkersClient | undefined;

  /**
   * Get the workers client for creating and managing workers
   * @returns A workers client instance
   */
  get workers() {
    if (!this._workers) {
      this._workers = new WorkersClient(this);
    }
    return this._workers;
  }

  /**
   * Get the API client for making HTTP requests to the Hatchet API
   * Note: This is not recommended for general use, but is available for advanced scenarios
   * @returns A API client instance
   */
  get api() {
    return this.v0.api;
  }

  /**
   * @deprecated use workflow.run, client.run, or client.* feature methods instead
   */
  get admin() {
    return this.v0.admin;
  }

  /**
   * Creates a new worker instance for processing workflow tasks.
   * @param options - Configuration options for creating the worker
   * @returns A promise that resolves with a new HatchetWorker instance
   */
  worker(name: string, options?: CreateWorkerOpts | number): Promise<Worker> {
    let opts: CreateWorkerOpts = {};
    if (typeof options === 'number') {
      opts = { slots: options };
    } else {
      opts = options || {};
    }

    return Worker.create(this.v0, name, opts);
  }

  /**
   * Register a webhook with the worker
   * @param workflows - The workflows to register on the webhooks
   * @returns A promise that resolves when the webhook is registered
   */
  webhooks(workflows: V0Workflow[]) {
    return this.v0.webhooks(workflows);
  }
}
